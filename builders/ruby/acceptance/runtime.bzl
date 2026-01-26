"""
Generated bzl file. Do not update manually.
We run acceptance tests only on full stacks.
"""

gae_runtimes = {
    "ruby25": "2.5.9",
    "ruby26": "2.6.10",
    "ruby27": "2.7.8",
    "ruby30": "3.0.7",
    "ruby32": "3.2.10",
    "ruby33": "3.3.10",
    "ruby34": "3.4.8",
}

gcf_runtimes = {
    "ruby26": "2.6.10",
    "ruby27": "2.7.8",
    "ruby30": "3.0.7",
    "ruby32": "3.2.10",
    "ruby33": "3.3.10",
    "ruby34": "3.4.8",
}

flex_runtimes = {
    "ruby32": "3.2.10",
    "ruby33": "3.3.10",
    "ruby34": "3.4.8",
}

version_to_stack = {
    "ruby25": "google-18-full",
    "ruby26": "google-18-full",
    "ruby27": "google-18-full",
    "ruby30": "google-18-full",
    "ruby32": "google-22-full",
    "ruby33": "google-22-full",
    "ruby34": "google-22-full",
}
