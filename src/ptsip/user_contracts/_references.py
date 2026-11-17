"""Close forward references against the supplied Policy v4 and RBAC documents."""

from ._schema import Issue, structure_issues


def _index(records, key, path, issues):
    indexed = {}
    duplicates = set()
    for position, record in enumerate(records):
        if key not in record:
            continue
        identifier = record[key]
        if identifier in indexed:
            duplicates.add(identifier)
            issues.append(Issue("DUPLICATE_ID", (*path, position, key), f"Duplicate ID: {identifier!r}"))
        else:
            indexed[identifier] = record
    for identifier in duplicates:
        del indexed[identifier]
    return indexed


def _references_to_conditions(records, kind, targets):
    ids_field = f"{kind}_ids"
    names_field = f"{kind}_name"
    for position, record in enumerate(records):
        prefix = ("RA_list", "entries", position)
        links = [next(iter(pair.items())) for pair in record.get(ids_field, [])]
        if names_field in record and record[names_field] != [name for name, _ in links]:
            yield Issue(
                "NAME_PAIRING", (*prefix, names_field),
                f"{kind} display names must match the names of the linked IDs in order",
            )
        seen = set()
        for index, (name, identifier) in enumerate(links):
            path = (*prefix, ids_field, index)
            if identifier in seen:
                yield Issue("DUPLICATE_ID", path, f"Duplicate {kind} reference: {identifier!r}")
            seen.add(identifier)
            target = targets.get(identifier)
            if target is None:
                yield Issue(
                    "UNKNOWN_ID", path,
                    f"No unique {kind} target for {identifier!r}",
                )
            elif target.get(names_field) != name:
                yield Issue(
                    "NAME_MISMATCH", path,
                    f"Display name differs from the owning {kind} record for {identifier!r}",
                )


def rbac_index(ra, rac, aec):
    issues = []
    for kind, document in (("RA", ra), ("RAC", rac), ("AEC", aec)):
        if document is not None:
            issues.extend(structure_issues(f"{kind}_list", document, (kind,)))
    if issues:
        return None, tuple(issues)

    records = (ra or {}).get("RA_list", {}).get("entries", [])
    indexed = _index(records, "RA_id", ("RA_list", "entries"), issues)
    for kind, document in (("AEC", aec), ("RAC", rac)):
        list_field = f"{kind}_list"
        targets = _index((document or {}).get(list_field, []), f"{kind}_id", (list_field,), issues)
        issues.extend(_references_to_conditions(records, kind, targets))
    return indexed, tuple(issues)


def validate_rbac(*, ra=None, rac=None, aec=None):
    """Validate the supplied list structures, IDs, and canonical-name references."""
    _, issues = rbac_index(ra, rac, aec)
    return issues


def policy_references(policy, rbac):
    issues = []
    usage = {}
    content = _index(
        policy.get("Policy_Content", {}).get("items", []), "id", ("Policy_Content", "items"), issues
    )
    for position, rule in enumerate(policy.get("Policy_Rules", {}).get("rules", [])):
        path = ("Policy_Rules", "rules", position)
        if "RA_id" in rule and rbac is not None:
            identifier = rule["RA_id"]
            if identifier in rbac:
                usage.setdefault(identifier, []).append((*path, "RA_id"))
            else:
                issues.append(Issue("UNKNOWN_ID", (*path, "RA_id"), f"No unique RA target for {identifier!r}"))
        for index, identifier in enumerate(rule.get("content", {}).get("content_id", [])):
            if identifier not in content:
                issues.append(Issue(
                    "UNKNOWN_ID", (*path, "content", "content_id", index),
                    f"No unique Content ID {identifier!r} in this policy document",
                ))
    # A reverse GUI lookup is a read-only projection of Policy Document usage.
    return tuple(issues), tuple((identifier, tuple(paths)) for identifier, paths in usage.items())
