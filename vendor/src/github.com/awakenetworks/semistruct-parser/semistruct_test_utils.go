package semistruct

import (
	"fmt"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"reflect"
	"strings"
	"unicode"
)

// Produce a bracketed attribute string given a map.
//
// Ex: map[string]string{ab:"age 22", an:bl} -> { ab="age 22" an=bl }
func mkAttrStr(m map[string]string, q bool) string {
	var acc []string
	for k, v := range m {
		if q {
			v = fmt.Sprint("\"", v, "\"")
		}
		acc = append(acc, fmt.Sprintf("%s=%s", k, v))
	}

	attrs := strings.Join(acc[:], " ")
	return fmt.Sprint("{ ", attrs, " }")
}

// Produce a tag list given a slice of strings.
//
// Ex: []string{"one", "two", "three"} -> [one:two:three]
func mkTagStr(t []string) string {
	tags := strings.Join(t[:], ":")
	return fmt.Sprint("[", tags, "]")
}

// Produce a parseable log line given a priority integer, a tag string
// and an attribute string.
func mkLogLine(priority int16, tags string, attrs string) string {
	return fmt.Sprintf("!< %d %s %s >!", priority, tags, attrs)
}

// Generate an arbitrary rune selected from the valid set of symbols,
// punctuation, and "other" runes.
//
// NOTE: this excludes the double quote character explicitly because
// we cannot parse it within the double quoted string, we can change
// the attribute value format if we need to to accomodate double
// quotes within the values, though.
//
// TODO: Gopter's primary example of combining the rune generators is
// with the Frequency function but I don't actually want to use that
// in order to compose all of these rune generators. Some more API
// spelunking is needed... This works for now, though.
func SpecialChar() gopter.Gen {
	return gen.Frequency(map[int]gopter.Gen{
		0: gen.RuneRange('#', '/'),
		1: gen.RuneRange(':', '?'),
		2: gen.RuneRange('{', '~'),
		3: gen.RuneRange('[', '_'),
		4: gen.RuneRange('@', '@'),
		5: gen.RuneRange('`', '`'),
	})
}

// TODO: figure out how to compose the two rune generators for the two
// specific characters we want (space and tab) - I'm doing it via the
// frequency generator and the RuneRange generator.
func WhiteSpaceChar() gopter.Gen {
	return gen.Frequency(map[int]gopter.Gen{
		// Generate a SPACE rune
		0: gen.RuneRange(32, 32),

		// Generate a TAB rune
		9: gen.RuneRange(9, 9),
	})
}

// Generate an arbitrary string of characters selected from the valid
// set of alphanum and special character runes.
//
// TODO: make sure the unicode character tests are complete and
// exhaustive for the sieve. This is important for complete property
// test coverage.
func AlphaNumSpecialString() gopter.Gen {
	return gopter.CombineGens(
		gen.SliceOf(gen.AlphaNumChar()),
		gen.SliceOf(SpecialChar()),
		gen.SliceOf(WhiteSpaceChar()),
	).Map(func(values []interface{}) string {
		v := append(values[0].([]rune), values[2].([]rune)...)
		v = append(v, values[1].([]rune)...)
		return string(v)
	}).SuchThat(func(str string) bool {
		for _, ch := range str {
			if !unicode.IsLetter(ch) &&
				!unicode.IsDigit(ch) &&
				!unicode.Is(unicode.Punct, ch) &&
				!unicode.Is(unicode.Symbol, ch) &&
				!unicode.Is(unicode.Mark, ch) &&
				!unicode.Is(unicode.Other, ch) &&
				!unicode.IsSpace(ch) {

				return false
			}
		}
		return true
	}).WithShrinker(gen.StringShrinker)
}

// UpperIdentifier generates an arbitrary identifier string.
//
// UpperIdentitiers are supposed to start with a letter and contain
// only uppercase letters, digits, and underscores
func UpperIdentifier() gopter.Gen {
	return gopter.CombineGens(
		gen.SliceOf(gen.AlphaUpperChar()),
		gen.SliceOf(gen.NumChar()),
		gen.SliceOf(gen.RuneRange('_', '_')),
	).Map(func(values []interface{}) string {
		v := append(values[0].([]rune), values[2].([]rune)...)
		v = append(v, values[1].([]rune)...)
		return string(v)
	}).SuchThat(func(str string) bool {
		if len(str) < 1 || !unicode.IsUpper(([]rune(str))[0]) {
			return false
		}
		for _, ch := range str {
			if !unicode.IsUpper(ch) && !unicode.IsDigit(ch) && !unicode.Is(unicode.Punct, ch) {
				return false
			}
		}
		return true
	}).WithShrinker(gen.StringShrinker)
}

// Because this isn't exported by gopter...grrr
func genString(runeGen gopter.Gen, runeSieve func(ch rune) bool) gopter.Gen {
	return gen.SliceOf(runeGen).Map(runesToString).SuchThat(func(v string) bool {
		for _, ch := range v {
			if !runeSieve(ch) {
				return false
			}
		}
		return true
	}).WithShrinker(gen.StringShrinker)
}

// Because this isn't exported by gopter...grrr
func runesToString(v []rune) string {
	return string(v)
}

// Generate an arbitrary map data structure given a key element
// generator and a value element generator. The number of pairs within
// the map is determined using gopter's internal Rng state.
//
// For a fixed-size generator of maps, please see MapOfN.
func MapOf(keyGen gopter.Gen, valGen gopter.Gen) gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		len := 0
		if genParams.Size > 0 {
			len = genParams.Rng.Intn(genParams.Size)
		}

		//result, keySieve, valSieve, keyShrinker, valShrinker := genMap(keyGen, valGen, genParams, len)
		result, _, _, _, _ := genMap(keyGen, valGen, genParams, len)

		genResult := gopter.NewGenResult(result.Interface(), gopter.NoShrinker)
		// if elementSieve != nil {
		// 	genResult.Sieve = forAllSieve(elementSieve)
		// }
		return genResult
	}
}

// Generate an arbitrary map data structure given a length argument
// specifying the fixed number of pairs we wish to be generated for
// the map and a key element generator and a value element generator.
//
// For a generator that produces arbitrarily-sized maps, please see
// MapOf.
func MapOfN(len int, keyGen gopter.Gen, valGen gopter.Gen) gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		//result, keySieve, valSieve, keyShrinker, valShrinker := genMap(keyGen, valGen, genParams, len)
		result, _, _, _, _ := genMap(keyGen, valGen, genParams, len)

		genResult := gopter.NewGenResult(result.Interface(), gopter.NoShrinker)
		// if elementSieve != nil {
		// 	genResult.Sieve = forAllSieve(elementSieve)
		// }
		return genResult
	}
}

// Generate a map given a key element generator, value element
// generator, and a length specifying the number of pairs we want the
// map to hold.
//
// This was ripped off of Gopter's genSlice function and expanded to
// accomodate the map data structure.
func genMap(keyGen gopter.Gen, valGen gopter.Gen, genParams *gopter.GenParameters, len int) (reflect.Value, func(interface{}) bool, func(interface{}) bool, gopter.Shrinker, gopter.Shrinker) {
	key := keyGen(genParams)
	val := valGen(genParams)

	result := reflect.MakeMap(reflect.MapOf(key.ResultType, val.ResultType))

	for i := 0; i < len; i++ {
		keyV, kok := key.Retrieve()
		valV, vok := val.Retrieve()

		if kok && vok {
			result.SetMapIndex(reflect.ValueOf(keyV), reflect.ValueOf(valV))
		}

		key = keyGen(genParams)
		val = valGen(genParams)
	}

	// TODO: I wonder if there's a better way to return all of these -
	// perhaps just returning the key and val itself would be nice so
	// we're not generating these big tuple return types that need to
	// be unpacked...
	return result, key.Sieve, val.Sieve, key.Shrinker, val.Shrinker
}
