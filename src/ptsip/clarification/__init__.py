"""Deterministic human clarification support for unresolved architectural intent."""

from .generator import ClarificationAnalysis, analyze_clarifications
from .i18n import SUPPORTED_LANGUAGES, resolve_language

__all__ = ["ClarificationAnalysis", "SUPPORTED_LANGUAGES", "analyze_clarifications", "resolve_language"]
