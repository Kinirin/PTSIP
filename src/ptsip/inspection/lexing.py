from __future__ import annotations


def code_positions(source: str, *, backtick_strings: bool = False) -> tuple[bool, ...]:
    """Return a same-length mask identifying code outside comments and strings.

    Newlines remain code positions so line-oriented parsers retain their original
    line accounting.  The scanner is intentionally bounded to the comment and
    string forms shared by Go, JavaScript/TypeScript, and C# dependency syntax.
    """

    mask = [True] * len(source)
    index = 0
    state = "code"
    quote = ""
    while index < len(source):
        char = source[index]
        following = source[index + 1] if index + 1 < len(source) else ""
        if state == "line-comment":
            if char == "\n":
                state = "code"
            else:
                mask[index] = False
            index += 1
            continue
        if state == "block-comment":
            mask[index] = False
            if char == "*" and following == "/":
                mask[index + 1] = False
                index += 2
                state = "code"
            else:
                index += 1
            continue
        if state == "string":
            mask[index] = False
            if quote != "`" and char == "\\":
                if index + 1 < len(source):
                    mask[index + 1] = False
                index += 2
                continue
            if char == quote:
                state = "code"
            index += 1
            continue

        if char == "/" and following == "/":
            mask[index] = mask[index + 1] = False
            index += 2
            state = "line-comment"
            continue
        if char == "/" and following == "*":
            mask[index] = mask[index + 1] = False
            index += 2
            state = "block-comment"
            continue
        if char in {"'", '"'} or (backtick_strings and char == "`"):
            mask[index] = False
            quote = char
            state = "string"
        index += 1
    return tuple(mask)


def keyword_is_code(source: str, mask: tuple[bool, ...], start: int, end: int, *keywords: str) -> bool:
    segment = source[start:end]
    offsets = [segment.find(keyword) for keyword in keywords]
    offsets = [offset for offset in offsets if offset >= 0]
    return bool(offsets) and mask[start + min(offsets)]


def comments_removed(source: str, *, backtick_strings: bool = False) -> str:
    """Replace comment characters with spaces while preserving strings/newlines."""

    result = list(source)
    index = 0
    state = "code"
    quote = ""
    while index < len(source):
        char = source[index]
        following = source[index + 1] if index + 1 < len(source) else ""
        if state == "line-comment":
            if char == "\n":
                state = "code"
            else:
                result[index] = " "
            index += 1
            continue
        if state == "block-comment":
            if char != "\n":
                result[index] = " "
            if char == "*" and following == "/":
                result[index] = result[index + 1] = " "
                index += 2
                state = "code"
            else:
                index += 1
            continue
        if state == "string":
            if quote != "`" and char == "\\":
                index += 2
                continue
            if char == quote:
                state = "code"
            index += 1
            continue
        if char == "/" and following == "/":
            result[index] = result[index + 1] = " "
            index += 2
            state = "line-comment"
            continue
        if char == "/" and following == "*":
            result[index] = result[index + 1] = " "
            index += 2
            state = "block-comment"
            continue
        if char in {"'", '"'} or (backtick_strings and char == "`"):
            quote = char
            state = "string"
        index += 1
    return "".join(result)
