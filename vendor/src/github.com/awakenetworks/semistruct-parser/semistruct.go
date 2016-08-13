package semistruct

import (
	"errors"
	p "github.com/andyleap/parser"
	"strconv"
)

// Log is the data structure representing the result of a successful
// semistructured log line parse.
type Log struct {
	Priority int64
	Tags     []string
	Attrs    map[string]string
}

type kvPair struct {
	key string
	val string
}

var (
	ErrPriority   = errors.New("failed getting priority")
	ErrTags       = errors.New("failed getting tags")
	ErrAttributes = errors.New("failed getting attributes")
)

// NewLogParser instantiates a parser composed of combinators for
// parsing a semistructured log line into the SemistructLog struct
// data type.
func NewLogParser() *p.Grammar {
	skipSpace := SkipSpace()

	o := p.And(
		// Parse opening sentinel "!<"
		OpenSentinel(),
		skipSpace,

		// Parse our log-level priority
		p.Tag("priority", PriorityInt()),
		skipSpace,

		// Parse our tag list "[tag1:tag2:tag3]"
		p.Tag("tags", Tags()),
		skipSpace,

		// Parse our attribute set "{ key=val key2=val2 }"
		p.Tag("attrs", Attrs()),
		skipSpace,

		// Parse the ending sentinel ">!"
		EndSentinel(),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		pr, ok := p.GetTag(m, "priority").(int64)
		if !ok {
			return nil, ErrPriority
		}

		tg, ok := p.GetTag(m, "tags").([]string)
		if !ok {
			return nil, ErrTags
		}

		at, ok := p.GetTag(m, "attrs").(map[string]string)
		if !ok {
			return nil, ErrAttributes
		}

		return &Log{pr, tg, at}, nil
	})

	return o
}

// PriorityInt parses the log level priority indicator; this can only
// be an integer from 0 to 9.
func PriorityInt() *p.Grammar {
	o := p.Set("0-9")
	o.Node(func(m p.Match) (p.Match, error) {
		v, err := strconv.ParseInt(p.String(m), 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	})

	return o
}

// AlphaNum parses zero or more characters that are alpha-numeric with
// an underscore.
func AlphaNum() *p.Grammar {
	return p.Mult(
		0, 0, p.Set(`\w\-`),
	)
}

// AlphaUpperNum parses zero or more characters that are uppercase
// alphabetic, numeric, and underscore.
func AlphaUpperNum() *p.Grammar {
	return p.Mult(
		0, 0, p.Set("A-Z0-9_"),
	)
}

// AlphaNumSpecial parses zero or more characters that are
// alpha-numeric, space characters, and special characters.
func AlphaNumSpecial() *p.Grammar {
	return p.Mult(
		0, 0,
		p.Set(`\w\-~ `+"`"+`\t!@#:;\$%^&\*\(\)\+=\?\\\/><,\.\{\}\[\]\|'`),
	)
}

// Alpha parses zero or more characters that are alphabetic and
// underscore.
func Alpha() *p.Grammar {
	return p.Mult(
		0, 0, p.Set("a-zA-Z_"),
	)
}

// QuotedStr parses an `AlphaNumSpecial` string wrapped in double
// quotes.
func QuotedStr() *p.Grammar {
	return p.And(
		p.Ignore(p.Lit(`"`)),
		AlphaNumSpecial(),
		p.Ignore(p.Lit(`"`)),
	)
}

// OpenSentinel consumes the opening sentinel "!<" character sequence.
func OpenSentinel() *p.Grammar {
	return p.Ignore(
		p.And(p.Lit("!"), p.Lit("<")),
	)
}

// EndSentinel consumes the ending sentinel ">!" character sequence.
func EndSentinel() *p.Grammar {
	return p.Ignore(
		p.And(p.Lit(">"), p.Lit("!")),
	)
}

// SkipSpace Consumes whitespace.
func SkipSpace() *p.Grammar {
	return p.Ignore(
		p.Mult(0, 0, p.Set(`\t `)),
	)
}

// Tag parses zero or more tag atoms (tag1:tag2) separated by a colon.
func Tag() *p.Grammar {
	alphaNum := AlphaNum()

	o := p.Optional(
		p.And(
			p.Tag("tag", alphaNum),
			p.Mult(
				0, 0, p.And(
					p.Ignore(p.Lit(":")),
					p.Tag("tag", alphaNum),
				),
			),
		),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		var tagAccum []string
		for _, op := range p.GetTags(m, "tag") {
			tagAccum = append(tagAccum, p.String(op))
		}
		return tagAccum, nil
	})
	return o
}

// Tags parses a high-level set of colon separated tag atoms bracketed
// by square brackets: [tag1:tag2].
func Tags() *p.Grammar {
	o := p.Optional(
		p.And(
			p.Ignore(p.Lit("[")),
			p.Tag("tag-elements", Tag()),
			p.Ignore(p.Lit("]")),
		),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		elems := p.GetTag(m, "tag-elements")
		if elems == nil {
			elems = []string{}
		}
		return elems, nil
	})
	return o
}

// Attrs parses a key value pair separated by an equal sign.
//
// Two types of values are supported by the parser: a quoted string
// value and an unquoted string value. The quoted string may contain
// spaces and many special characters.
//
// The unquoted value string may contain: a-zA-Z0-9_-
//
// The unquoted version should be able to contain the other special
// characters too but supporting that or not wasn't specified so I
// kept the unquoted version for very simple strings.
//
// Order within the Or combinator is important here, we want to try
// matching the quoted AlphaNumSpecial value first before then attempt
// the unquoted plain AlphaNum value.
func Attrs() *p.Grammar {
	o := p.Optional(
		p.And(
			p.Ignore(p.Lit("{")),
			SkipSpace(),
			p.Tag("attrmap", KvPairs()),
			SkipSpace(),
			p.Ignore(p.Lit("}")),
		),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		attrmap := p.GetTag(m, "attrmap")
		if attrmap == nil {
			attrmap = make(map[string]string)
		}

		return attrmap, nil
	})

	return o
}

// KvPair parses a key value pair that is right-aligned and separated
// by an equal sign.
func KvPair() *p.Grammar {
	o := p.And(
		p.Tag("key", p.And(p.Set("A-Z0-9"), AlphaUpperNum())),
		p.Lit("="),
		p.Tag(
			"value",
			p.Or(QuotedStr(), AlphaNum()),
		),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		k := p.String(p.GetTag(m, "key"))
		v := p.String(p.GetTag(m, "value"))
		return kvPair{k, v}, nil
	})
	return o
}

// KvPairs parses zero or more key value pairs bracketed by curly
// brackets.
func KvPairs() *p.Grammar {
	kvPairParser := KvPair()

	o := p.Optional(
		p.And(
			p.Tag("pair", kvPairParser),
			p.Mult(0, 0,
				p.And(
					SkipSpace(),
					p.Tag("pair", kvPairParser),
				),
			),
		),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		attrmap := make(map[string]string)
		for _, v := range p.GetTags(m, "pair") {
			kvp := v.(kvPair)
			attrmap[kvp.key] = kvp.val
		}
		return attrmap, nil
	})
	return o
}
