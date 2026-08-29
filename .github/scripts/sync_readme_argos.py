from __future__ import annotations

import argparse
import difflib
import re
import subprocess
from collections import defaultdict, deque
from pathlib import Path
from typing import Protocol

SOURCE_LANGUAGE = "en"
GENERATED_COMMENT = (
    "<!-- AUTO-GENERATED: README.md is canonical. "
    "Unchanged reviewed translations are preserved; changed source blocks are "
    "translated by self-hosted Argos Translate. Do not edit directly. -->"
)
LANGUAGE_METADATA = {
    "ko": {
        "label": "한국어",
        "notice": (
            "> 이 문서는 정식 원본인 [`README.md`](README.md)를 기준으로 유지되는 "
            "한국어 번역본입니다. 변경되지 않은 번역은 보존하고, 변경된 원문 블록만 "
            "self-hosted Argos Translate로 갱신합니다. 프로젝트 사실이나 의미가 상충할 "
            "경우 영문 원본을 기준으로 합니다."
        ),
    },
}
NORMATIVE = ("MUST NOT", "SHOULD NOT", "MUST", "SHOULD", "MAY")
PROPER_TERMS = (
    "Primary Lifecycle Ownership and Responsibility Isolation Policy",
    "Product SDK Plane",
    "Toolchain SDK Plane",
    "Consumer Repository",
    "Reference Tool",
    "Enforced Conformance",
    "Responsibility Map",
    "Project Profile",
    "Product Artifact",
    "Specification",
    "PTSIP",
    "VPMS",
    "Pilot",
    "Tool",
)
PROTECTED = re.compile(
    r"(`+[^`\n]*`+)"
    r"|(\[[^\]\n]+\]\([^)]+\))"
    r"|(<[^>\n]+>)"
    r"|(https?://[^\s<>)]+)"
    r"|(\b[0-9a-f]{40,64}\b)"
    r"|(\bv?\d+\.\d+(?:\.\d+)?(?:-[A-Za-z0-9.]+)?\b)"
    r"|(\*\*|__|~~)"
    r"|\b(" + "|".join(re.escape(x) for x in (*NORMATIVE, *PROPER_TERMS)) + r")\b"
)
FENCE = re.compile(r"^\s*(```+|~~~+)")
TABLE_DIVIDER = re.compile(r"^\s*:?-{3,}:?\s*$")
LIST_PREFIX = re.compile(r"^(\s*(?:[-+*]|\d+[.)])\s+)(.*)$")
HEADING_PREFIX = re.compile(r"^(#{1,6}\s+)(.*)$")
QUOTE_PREFIX = re.compile(r"^(>\s*)(.*)$")


class Translator(Protocol):
    def translate(self, text: str) -> str: ...


class ArgosTranslator:
    def __init__(self, source: str, target: str) -> None:
        import argostranslate.package as package
        import argostranslate.translate as translate
        from packaging.version import Version

        installed = package.get_installed_packages()
        direct = any(
            pkg.type == "translate"
            and pkg.from_code == source
            and pkg.to_code == target
            for pkg in installed
        )
        if not direct:
            print(f"Installing Argos model {source}->{target}.")
            package.update_package_index()
            available = [
                pkg
                for pkg in package.get_available_packages()
                if pkg.type == "translate"
                and pkg.from_code == source
                and pkg.to_code == target
            ]
            if not available:
                raise RuntimeError(
                    f"No Argos Translate model is available for {source}->{target}."
                )
            selected = max(
                available,
                key=lambda pkg: Version(pkg.package_version or "0"),
            )
            package.install_from_path(selected.download())
            print(
                f"Installed Argos model {source}->{target} "
                f"{selected.package_version or 'unknown'}."
            )

        loaded = translate.get_translation_from_codes(source, target)
        if loaded is None:
            raise RuntimeError(f"Unable to load Argos translation {source}->{target}.")
        self._translation = loaded

    def translate(self, text: str) -> str:
        result = self._translation.translate(text)
        if not isinstance(result, str) or not result.strip():
            raise RuntimeError("Argos Translate returned an empty segment.")
        return result


def strip_navigation(text: str) -> str:
    return re.sub(
        r'\A\s*<p\s+align=["\']right["\']>.*?</p>\s*',
        "",
        text,
        count=1,
        flags=re.DOTALL | re.IGNORECASE,
    )


def strip_localized_prefix(text: str) -> str:
    text = text.lstrip("\ufeff")
    text = re.sub(r"\A\s*<!--.*?-->\s*", "", text, count=1, flags=re.DOTALL)
    text = strip_navigation(text)
    blocks = split_blocks(text)
    if (
        len(blocks) >= 2
        and blocks[0].startswith("# ")
        and blocks[1].startswith(">")
        and "README.md" in blocks[1]
    ):
        del blocks[1]
    return "\n\n".join(blocks).rstrip() + "\n"


def split_blocks(text: str) -> list[str]:
    blocks: list[str] = []
    current: list[str] = []
    in_fence = False
    fence_char = ""

    for line in text.replace("\r\n", "\n").split("\n"):
        fence = FENCE.match(line)
        if fence:
            marker = fence.group(1)[0]
            if not in_fence:
                in_fence = True
                fence_char = marker
            elif marker == fence_char:
                in_fence = False
                fence_char = ""
            current.append(line)
            continue

        if not line.strip() and not in_fence:
            if current:
                blocks.append("\n".join(current).rstrip())
                current = []
            continue

        current.append(line)

    if current:
        blocks.append("\n".join(current).rstrip())
    return blocks


def block_type(block: str) -> str:
    first = block.splitlines()[0].lstrip() if block.splitlines() else ""
    if FENCE.match(first):
        return "fence"
    if first.startswith("#"):
        return "heading"
    if first.startswith("|"):
        return "table"
    if first.startswith(">"):
        return "quote"
    if LIST_PREFIX.match(first):
        return "list"
    return "paragraph"


def git_text(ref: str, path: Path) -> str:
    completed = subprocess.run(
        ["git", "show", f"{ref}:{path.as_posix()}"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    if completed.returncode != 0:
        raise RuntimeError(
            f"Unable to read {path} from baseline {ref}: {completed.stderr.strip()}"
        )
    return completed.stdout.lstrip("\ufeff")


def baseline_ref(source_path: Path) -> str | None:
    change = subprocess.run(
        ["git", "log", "-1", "--format=%H", "--", source_path.as_posix()],
        check=True,
        stdout=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    ).stdout.strip()
    if not change:
        return None
    parents = subprocess.run(
        ["git", "rev-list", "--parents", "-n", "1", change],
        check=True,
        stdout=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    ).stdout.strip().split()
    return parents[1] if len(parents) >= 2 else None


def aligned_baseline(
    ref: str,
    source_path: Path,
    target_path: Path,
) -> tuple[list[str], list[str]]:
    source_blocks = split_blocks(strip_navigation(git_text(ref, source_path)))
    localized_blocks = split_blocks(strip_localized_prefix(git_text(ref, target_path)))
    if len(source_blocks) != len(localized_blocks):
        raise RuntimeError(
            "Incremental translation baseline is not structurally aligned: "
            f"{len(source_blocks)} source blocks != {len(localized_blocks)} localized blocks. "
            "Refusing a full-document overwrite."
        )
    for index, (source_block, localized_block) in enumerate(
        zip(source_blocks, localized_blocks, strict=True)
    ):
        source_type = block_type(source_block)
        localized_type = block_type(localized_block)
        if source_type != localized_type:
            raise RuntimeError(
                "Incremental translation baseline block types differ at "
                f"index {index}: {source_type} != {localized_type}. "
                "Refusing a full-document overwrite."
            )
    return source_blocks, localized_blocks


def translate_natural(text: str, translator: Translator) -> str:
    if not re.search(r"[A-Za-z]", text):
        return text

    saved: list[str] = []

    def replace(match: re.Match[str]) -> str:
        token = f"ZXQKEEP{len(saved):04d}QXZ"
        saved.append(match.group(0))
        return token

    protected = PROTECTED.sub(replace, text)
    translated = translator.translate(protected)
    for index, original in enumerate(saved):
        token = f"ZXQKEEP{index:04d}QXZ"
        if token not in translated:
            raise RuntimeError(
                f"Argos changed protected placeholder {token}; refusing unsafe output."
            )
        translated = translated.replace(token, original)
    return translated


def translate_table_line(line: str, translator: Translator) -> str:
    cells = line.split("|")
    cores = [cell.strip() for cell in cells[1:-1]]
    if cores and all(TABLE_DIVIDER.fullmatch(core) for core in cores):
        return line

    out: list[str] = []
    for cell in cells:
        leading = cell[: len(cell) - len(cell.lstrip())]
        trailing = cell[len(cell.rstrip()) :]
        core = cell.strip()
        if core:
            core = translate_natural(core, translator)
        out.append(f"{leading}{core}{trailing}")
    return "|".join(out)


def translate_line(line: str, translator: Translator) -> str:
    if not line.strip():
        return line
    if FENCE.match(line):
        return line
    heading = HEADING_PREFIX.match(line)
    if heading:
        if heading.group(1) == "# ":
            return line
        return heading.group(1) + translate_natural(heading.group(2), translator)
    quote = QUOTE_PREFIX.match(line)
    if quote:
        return quote.group(1) + translate_natural(quote.group(2), translator)
    item = LIST_PREFIX.match(line)
    if item:
        return item.group(1) + translate_natural(item.group(2), translator)
    if line.strip().startswith("|") and line.strip().endswith("|"):
        return translate_table_line(line, translator)
    if line.lstrip().startswith("<") and line.rstrip().endswith(">"):
        return line
    return translate_natural(line, translator)


def translate_block(block: str, translator: Translator) -> str:
    kind = block_type(block)
    if kind == "fence" or block.startswith("# "):
        return block
    return "\n".join(translate_line(line, translator) for line in block.splitlines())


def translate_changed_block(
    current: str,
    previous_source: str,
    previous_localized: str,
    translator: Translator,
) -> str:
    if block_type(current) == "fence":
        return current
    if current.startswith("# "):
        return current

    old_source_lines = previous_source.splitlines()
    old_localized_lines = previous_localized.splitlines()
    current_lines = current.splitlines()
    if len(old_source_lines) != len(old_localized_lines):
        return translate_block(current, translator)

    matcher = difflib.SequenceMatcher(a=old_source_lines, b=current_lines, autojunk=False)
    output: list[str] = []
    for tag, i1, i2, j1, j2 in matcher.get_opcodes():
        if tag == "equal":
            output.extend(old_localized_lines[i1:i2])
            continue
        if tag in {"replace", "insert"}:
            output.extend(translate_line(line, translator) for line in current_lines[j1:j2])
    return "\n".join(output)


def incremental_body(
    source: str,
    target_code: str,
    source_path: Path,
    target_path: Path,
    translator: Translator,
) -> tuple[str, int, int]:
    current_blocks = split_blocks(strip_navigation(source))
    ref = baseline_ref(source_path)
    if ref is None:
        translated = [translate_block(block, translator) for block in current_blocks]
        return "\n\n".join(translated), 0, len(translated)

    baseline_source, baseline_localized = aligned_baseline(ref, source_path, target_path)
    reusable: dict[str, deque[str]] = defaultdict(deque)
    for source_block, localized_block in zip(
        baseline_source, baseline_localized, strict=True
    ):
        reusable[source_block].append(localized_block)

    output: list[str] = []
    reused = 0
    translated_count = 0
    for index, current in enumerate(current_blocks):
        candidates = reusable.get(current)
        if candidates:
            output.append(candidates.popleft())
            reused += 1
            continue

        previous_source = baseline_source[index] if index < len(baseline_source) else None
        previous_localized = (
            baseline_localized[index] if index < len(baseline_localized) else None
        )
        if (
            previous_source is not None
            and previous_localized is not None
            and block_type(previous_source) == block_type(current)
            and difflib.SequenceMatcher(
                a=previous_source, b=current, autojunk=False
            ).ratio()
            >= 0.35
        ):
            output.append(
                translate_changed_block(
                    current,
                    previous_source,
                    previous_localized,
                    translator,
                )
            )
        else:
            output.append(translate_block(current, translator))
        translated_count += 1

    print(
        f"Incremental {target_code} translation baseline {ref}: "
        f"reused {reused} blocks; updated {translated_count} blocks."
    )
    return "\n\n".join(output), reused, translated_count


def render(
    source: str,
    target: str,
    source_path: Path,
    target_path: Path,
    translator: Translator,
) -> str:
    metadata = LANGUAGE_METADATA.get(target)
    if metadata is None:
        raise RuntimeError(f"Unsupported localized README language: {target}")

    body, _, _ = incremental_body(
        source,
        target,
        source_path,
        target_path,
        translator,
    )
    blocks = split_blocks(body)
    if not blocks or not blocks[0].startswith("# "):
        raise RuntimeError("README.md must contain a top-level '# ' title.")
    title = blocks[0]
    body_without_title = "\n\n".join(blocks[1:]).rstrip()

    nav = (
        '<p align="right">\n'
        '  <a href="README.md">English</a> | '
        f'{metadata["label"]}\n'
        "</p>"
    )
    return (
        f"{GENERATED_COMMENT}\n{nav}\n\n{title}\n\n"
        f"{metadata['notice']}\n\n{body_without_title}\n"
    )


def fenced_blocks(text: str) -> list[str]:
    return re.findall(r"(?ms)^```[^\n]*\n.*?^```[ \t]*$", text)


def heading_levels(text: str) -> list[int]:
    return [len(m.group(1)) for m in re.finditer(r"(?m)^(#{1,6})\s+", text)]


def link_targets(text: str) -> list[str]:
    return [x.strip().split(" ", 1)[0] for x in re.findall(r"\]\(([^)]+)\)", text)]


def inline_code(text: str) -> list[str]:
    return re.findall(r"`+[^`\n]*`+", text)


def validate(source: str, localized: str, target: str) -> None:
    canonical = strip_navigation(source)
    comparable_localized = strip_localized_prefix(localized)
    if target == "ko" and not re.search(r"[가-힣]", localized):
        raise RuntimeError("Translation validation failed: Korean text is missing.")
    if heading_levels(canonical) != heading_levels(comparable_localized):
        raise RuntimeError("Translation validation failed: heading structure changed.")
    if fenced_blocks(canonical) != fenced_blocks(comparable_localized):
        raise RuntimeError("Translation validation failed: fenced code blocks changed.")

    missing_links = [x for x in link_targets(canonical) if x not in localized]
    if missing_links:
        raise RuntimeError(
            "Translation validation failed: link targets disappeared: "
            + ", ".join(sorted(set(missing_links)))
        )

    missing_code = [x for x in inline_code(canonical) if x not in localized]
    if missing_code:
        raise RuntimeError(
            "Translation validation failed: inline code changed: "
            + ", ".join(sorted(set(missing_code))[:20])
        )

    for keyword in NORMATIVE:
        if re.search(rf"\b{re.escape(keyword)}\b", canonical) and keyword not in localized:
            raise RuntimeError(
                f"Translation validation failed: normative keyword {keyword!r} disappeared."
            )


def parse_target(value: str) -> tuple[str, Path]:
    if ":" not in value:
        raise argparse.ArgumentTypeError("target must use <language-code>:<path>")
    code, raw_path = value.split(":", 1)
    if not code.strip() or not raw_path.strip():
        raise argparse.ArgumentTypeError("target must use <language-code>:<path>")
    return code.strip(), Path(raw_path.strip())


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, default=Path("README.md"))
    parser.add_argument("--target", action="append", type=parse_target, required=True)
    args = parser.parse_args()

    source = args.source.read_text(encoding="utf-8").lstrip("\ufeff")
    for target_code, target_path in args.target:
        translator = ArgosTranslator(SOURCE_LANGUAGE, target_code)
        localized = render(
            source,
            target_code,
            args.source,
            target_path,
            translator,
        )
        validate(source, localized, target_code)

        current = (
            target_path.read_text(encoding="utf-8") if target_path.exists() else None
        )
        if current == localized:
            print(f"{target_path} is already synchronized.")
            continue
        target_path.write_text(localized, encoding="utf-8", newline="\n")
        print(f"Updated {target_path} with incremental local Argos Translate.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
