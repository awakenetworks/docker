# Welcome!

`semistruct-parse` is a project providing a parser for semistructured lines of
text. These are primarily used in a custom journald logging driver for the
purpose of feeding semistructured data into journald from docker.

## Testing

You need the following dependencies in order to run the test suite:

- `github.com/leanovate/gopter`
- `github.com/andyleap/parser`

There is a small hand-ful of hand-written unit tests for the log lines we
expected to be emitting and then a suite of property tests for each field of the
semistructured log line, using `gopter` for the framework.

## The Format

Here is an example of a semistructured line that, when parsed by the top-level
`ParseSemistruct` combinator, emits a struct of standard Go data structures
representing the three primary fields:

`!< 8 [db3:slave] { region="us-east-d z2" uuid=3c79bfc0-4b27-4c99-a952-8f435bbf0514 } >!`

The parser looks for the beginning sentinel character sequence `!<` and finishes
when it finds the ending sentinel character sequence `>!`.

The three primary types are a single-digit log level priority indicator, a list
of tags separated by a single colon (these may be alphanum with an underscore
and a hyphen), and an unordered container or map of key value attribute
pairs. The attribute key has the same character constraints as the tag values
(alphanum with an underscore and a hyphen) and there are two types of attribute
values supported:

1. Unquoted with the same character constraints as the keys
2. Quoted with the following character classes allowed:
  - Alphanum
  - All special characters _except for_ the double quote
  - Whitespace characters
  
The attribute keys must follow the journald constraints which means it must
begin with a letter and contain only uppercase letters, digits, and underscores.
