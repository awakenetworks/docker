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

`!< 8 [db3:slave] { REGION="us-east-d z2" UUID="3c79bfc0-4b27-4c99-a952-8f435bbf0514" } >!`

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

## TODO

- [ ] Stop supporting the non-double quoted value in attributes
- [ ] Convert the attribute syntax over to valid JSON
- [ ] Axe the combinator library in favor of a more performant parser, currently
  performance is the single biggest problem
  
### Performance Notes for Posterity

```
(pprof) top20
1950ms of 2990ms total (65.22%)
Dropped 42 nodes (cum <= 14.95ms)
Showing top 20 nodes out of 121 (cum >= 40ms)
      flat  flat%   sum%        cum   cum%
     290ms  9.70%  9.70%      570ms 19.06%  runtime.mallocgc
     250ms  8.36% 18.06%      250ms  8.36%  runtime.futex
     240ms  8.03% 26.09%      300ms 10.03%  runtime.updatememstats
     130ms  4.35% 30.43%      130ms  4.35%  runtime.heapBitsSetType
     120ms  4.01% 34.45%      250ms  8.36%  runtime.makeslice
     110ms  3.68% 38.13%      300ms 10.03%  runtime.growslice
     100ms  3.34% 41.47%      170ms  5.69%  regexp.(*machine).tryBacktrack
      90ms  3.01% 44.48%       90ms  3.01%  runtime.getitab
      90ms  3.01% 47.49%      270ms  9.03%  strconv.quoteWith
      60ms  2.01% 49.50%     2090ms 69.90%  github.com/andyleap/parser.And.func1
      60ms  2.01% 51.51%       60ms  2.01%  runtime.(*mspan).sweep.func1
      50ms  1.67% 53.18%      130ms  4.35%  github.com/andyleap/parser.GetTags
      50ms  1.67% 54.85%       50ms  1.67%  runtime.memmove
      50ms  1.67% 56.52%      290ms  9.70%  runtime.newarray
      50ms  1.67% 58.19%       50ms  1.67%  runtime.purgecachedstats
      50ms  1.67% 59.87%       50ms  1.67%  runtime/internal/atomic.Xchg
      40ms  1.34% 61.20%     1980ms 66.22%  github.com/andyleap/parser.Mult.func1
      40ms  1.34% 62.54%      180ms  6.02%  github.com/andyleap/parser.String
      40ms  1.34% 63.88%       40ms  1.34%  regexp.(*bitState).reset
      40ms  1.34% 65.22%       40ms  1.34%  runtime.prefetchnta
```

```
(pprof) top20 -cum
0.54s of 2.99s total (18.06%)
Dropped 42 nodes (cum <= 0.01s)
Showing top 20 nodes out of 121 (cum >= 0.35s)
      flat  flat%   sum%        cum   cum%
         0     0%     0%      2.80s 93.65%  runtime.goexit
     0.01s  0.33%  0.33%      2.68s 89.63%  _/media/parnell/Tardis/Programming/work/awakenetworks/semistruct-parser.BenchmarkParserSmall
         0     0%  0.33%      2.68s 89.63%  testing.(*B).launch
         0     0%  0.33%      2.68s 89.63%  testing.(*B).runN
         0     0%  0.33%      2.11s 70.57%  github.com/andyleap/parser.(*Grammar).ParseString
     0.01s  0.33%  0.67%      2.10s 70.23%  github.com/andyleap/parser.(*Grammar).Node.func1
     0.06s  2.01%  2.68%      2.09s 69.90%  github.com/andyleap/parser.And.func1
     0.04s  1.34%  4.01%      1.98s 66.22%  github.com/andyleap/parser.Mult.func1
     0.02s  0.67%  4.68%      1.74s 58.19%  github.com/andyleap/parser.Tag.func1
     0.03s  1.00%  5.69%      1.16s 38.80%  github.com/andyleap/parser.Set.func1
     0.01s  0.33%  6.02%      0.74s 24.75%  runtime.systemstack
     0.01s  0.33%  6.35%      0.62s 20.74%  github.com/andyleap/parser.Ignore.func1
     0.29s  9.70% 16.05%      0.57s 19.06%  runtime.mallocgc
         0     0% 16.05%      0.54s 18.06%  runtime.ReadMemStats
         0     0% 16.05%      0.42s 14.05%  fmt.Errorf
         0     0% 16.05%      0.41s 13.71%  fmt.Sprintf
     0.02s  0.67% 16.72%      0.39s 13.04%  regexp.(*Regexp).Match
     0.01s  0.33% 17.06%      0.37s 12.37%  regexp.(*Regexp).doExecute
     0.03s  1.00% 18.06%      0.36s 12.04%  fmt.(*pp).doPrintf
         0     0% 18.06%      0.35s 11.71%  github.com/andyleap/parser.Or.func1
```
