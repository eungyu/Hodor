package util

import (
  "fmt" 
  "log"
  "bytes"
  "errors"
  "strings"
  "io"
  "encoding/base64"
  "io/ioutil"
  "strconv"
)

var debug = debugT(false)

type debugT bool

func (d debugT) Printf(format string, args ...interface{}) {
	if d {
		log.Printf(format, args...)
	}
}

type QuotedPrintable []byte

type qDecoder struct {
	r       io.Reader
	scratch [2]byte
}

func (qd qDecoder) Read(p []byte) (n int, err error) {
	// This method writes at most one byte into p.
	if len(p) == 0 {
		return 0, nil
	}
	if _, err := qd.r.Read(qd.scratch[:1]); err != nil {
		return 0, err
	}
	switch c := qd.scratch[0]; {
	case c == '=':
		if _, err := io.ReadFull(qd.r, qd.scratch[:2]); err != nil {
			return 0, err
		}
		x, err := strconv.ParseInt(string(qd.scratch[:2]), 16, 64)
		if err != nil {
			return 0, fmt.Errorf("mail: invalid RFC 2047 encoding: %q", qd.scratch[:2])
		}
		p[0] = byte(x)
	case c == '_':
		p[0] = ' '
	default:
		p[0] = c
	}
	return 1, nil
}

func (p *QuotedPrintable) skipSpace() {
	*p = bytes.TrimLeft(*p, " \t")
}

func (p *QuotedPrintable) peek() byte {
	return (*p)[0]
}

func (p *QuotedPrintable) empty() bool {
	return p.len() == 0
}

func (p *QuotedPrintable) len() int {
	return len(*p)
}

var atextChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"abcdefghijklmnopqrstuvwxyz" +
	"0123456789" +
	"!#$%&'*+-/=?^_`{|}~")

// isAtext returns true if c is an RFC 5322 atext character.
// If dot is true, period is included.
func isAtext(c byte, dot bool) bool {
	if dot && c == '.' {
		return true
	}
	return bytes.IndexByte(atextChars, c) >= 0
}

// isQtext returns true if c is an RFC 5322 qtest character.
func isQtext(c byte) bool {
	// Printable US-ASCII, excluding backslash or quote.
	if c == '\\' || c == '"' {
		return false
	}
	return '!' <= c && c <= '~'
}

// consumeAtom parses an RFC 5322 atom at the start of p.
// If dot is true, consumeAtom parses an RFC 5322 dot-atom instead.
func (p *QuotedPrintable) consumeAtom(dot bool) (atom string, err error) {
	if !isAtext(p.peek(), false) {
		return "", errors.New("mail: invalid string")
	}
	i := 1
	for ; i < p.len() && isAtext((*p)[i], dot); i++ {
	}
	atom, *p = string((*p)[:i]), (*p)[i:]
	return atom, nil
}

// consumePhrase parses the RFC 5322 phrase at the start of p.
func (p *QuotedPrintable) Decode() (phrase string, err error) {
	debug.Printf("consumePhrase: [%s]", *p)
	// phrase = 1*word
	var words []string
	for {
		// word = atom / QuotedPrintable-string
		var word string
		p.skipSpace()
		if p.empty() {
			if len(words) == 0 {
				return "", errors.New("mail: missing phrase")
			} else {
				break;
			}
		}
		word, err = p.consumeAtom(false)

		// RFC 2047 encoded-word starts with =?, ends with ?=, and has two other ?s.
		if err == nil && strings.HasPrefix(word, "=?") && strings.HasSuffix(word, "?=") && strings.Count(word, "?") == 4 {
			word, err = decodeRFC2047Word(word)
		}

		if err != nil {
			break
		}
		debug.Printf("consumePhrase: consumed %q", word)
		words = append(words, word)
	}
	// Ignore any error if we got at least one word.
	if err != nil && len(words) == 0 {
		debug.Printf("consumePhrase: hit err: %v", err)
		return "", errors.New("mail: missing word in phrase")
	}

	phrase = strings.Join(words, " ")
	return phrase, nil
}

func decodeRFC2047Word(s string) (string, error) {
	fields := strings.Split(s, "?")
	if len(fields) != 5 || fields[0] != "=" || fields[4] != "=" {
		return "", errors.New("mail: address not RFC 2047 encoded")
	}
	charset, enc := strings.ToLower(fields[1]), strings.ToLower(fields[2])
	if charset != "iso-8859-1" && charset != "utf-8" {
		return "", fmt.Errorf("mail: charset not supported: %q", charset)
	}

	in := bytes.NewBufferString(fields[3])
	var r io.Reader
	switch enc {
	case "b":
		r = base64.NewDecoder(base64.StdEncoding, in)
	case "q":
		r = qDecoder{r: in}
	default:
		return "", fmt.Errorf("mail: RFC 2047 encoding not supported: %q", enc)
	}

	dec, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	switch charset {
	case "iso-8859-1":
		b := new(bytes.Buffer)
		for _, c := range dec {
			b.WriteRune(rune(c))
		}
		return b.String(), nil
	case "utf-8":
		return string(dec), nil
	}
	panic("unreachable")
}
