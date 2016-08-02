package semistruct

import "strconv"
import p "github.com/andyleap/parser"

type Semistruct_log struct {
	priority int64
	tags     []string
	attrs    map[string]string
}

type kv_pair struct {
	key string
	val string
}

// Parse a semistructured log line into the semistruct_log struct
// type.
func ParseSemistruct() *p.Grammar {
	o := p.And(
		// Parse opening sentinel "!<"
		OpenSentinel(),
		SkipSpace(),

		// Parse our log-level priority
		p.Tag("priority", PriorityInt()),
		SkipSpace(),

		// Parse our tag list "[tag1:tag2:tag3]"
		p.Tag("tags", Tags()),
		SkipSpace(),

		// Parse our attribute set "{ key=val key2=val2 }"
		p.Tag("attrs", Attrs()),
		SkipSpace(),

		// Parse the ending sentinel ">!"
		EndSentinel(),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		pr := p.GetTag(m, "priority").(int64)
		tg := p.GetTag(m, "tags").([]string)
		at := p.GetTag(m, "attrs").(map[string]string)

		return Semistruct_log{pr, tg, at}, nil
	})

	return o
}

// TODO: Make sure all fields in the schema that are optional are
// handled in here with good mempty-like defaults (or is there an
// option type analog in Go?).
//
// TODO: Rename some of these parser combinators.
//
// TODO: Refactor the repetitive Mult calls all over the place if I
// can, there's a lot of repetition I'd like to get rid of on my first
// pass.
//
// TODO: handle failure better - if the string can't parse, what do we
// do?
//
// TODO: are we consuming to the end of the line???

// Parse the log level priority indicator; this can only be an integer
// from 0 to 9
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

// Parse zero or more characters that are alpha-numeric (note, \w
// includes the underscore).
func AlphaNum() *p.Grammar {
	return p.Mult(
		0, 0, p.Set("\\w\\-"),
	)
}

// Parse zero or more characters that are alpha-numeric (note, \w
// includes the underscore).
func AlphaUpperNum() *p.Grammar {
	return p.Mult(
		0, 0, p.Set("A-Z0-9_"),
	)
}

// Parse zero or more characters that are alpha-numeric, space
// characters, and special characters (note, \w includes the
// underscore).
func AlphaNumSpecial() *p.Grammar {
	return p.Mult(
		0, 0,
		p.Set("\\w\\-~ `\\t!@#:;\\$%^&\\*\\(\\)\\+=\\?\\\\/><,\\.\\{\\}\\[\\]\\|'"),
	)
}

// Parse zero or more characters that are alpha and underscore.
func Alpha() *p.Grammar {
	return p.Mult(
		0, 0, p.Set("a-zA-Z_"),
	)
}

// Parse a alphaSpecial string wrapped in double quotes.
func QuotedStr() *p.Grammar {
	return p.And(
		p.Ignore(p.Lit("\"")),
		AlphaNumSpecial(),
		p.Ignore(p.Lit("\"")),
	)
}

// Consume the opening sentinel "!<".
func OpenSentinel() *p.Grammar {
	return p.Ignore(
		p.And(p.Lit("!"), p.Lit("<")),
	)
}

// Consume the ending sentinel ">!".
func EndSentinel() *p.Grammar {
	return p.Ignore(
		p.And(p.Lit(">"), p.Lit("!")),
	)
}

// Consume whitespace.
func SkipSpace() *p.Grammar {
	return p.Ignore(
		p.Mult(0, 0, p.Set("\\t ")),
	)
}

// Parse zero or more tag atoms separated by a colon.
func Tag() *p.Grammar {
	o := p.Optional(
		p.And(
			p.Tag("tag", AlphaNum()),
			p.Mult(
				0, 0, p.And(
					p.Ignore(p.Lit(":")),
					p.Tag("tag", AlphaNum()),
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
			elems = make([]string, 0)
		}
		return elems, nil
	})
	return o
}

// Parse a key value pair separated by an equal sign.
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
// matching the quoted alphaNumSpecial value first before then attempt
// the unquoted plain alphaNum value.
func Attrs() *p.Grammar {
	o := p.Optional(
		p.And(
			p.Ignore(p.Lit("{")),
			SkipSpace(),
			p.Tag("attrmap", Kvpairs()),
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

func Kvpair() *p.Grammar {
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
		return kv_pair{k, v}, nil
	})
	return o
}

func Kvpairs() *p.Grammar {
	o := p.Optional(
		p.And(
			p.Tag("pair", Kvpair()),
			p.Mult(0, 0,
				p.And(
					SkipSpace(),
					p.Tag("pair", Kvpair()),
				),
			),
		),
	)

	o.Node(func(m p.Match) (p.Match, error) {
		attrmap := make(map[string]string)
		for _, v := range p.GetTags(m, "pair") {
			kvp := v.(kv_pair)
			attrmap[kvp.key] = kvp.val
		}
		return attrmap, nil
	})
	return o
}
