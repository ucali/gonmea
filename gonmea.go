package nmea

import (
	"bytes"
	//"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	STATE_OPEN     = 1
	STATE_CLOSED   = 2
	STATE_CHECKING = 3
)

// Parser

type Parser struct {
	State    int
	Sentence string
	Count    uint64
	Output   chan string

	checksum         byte
	expectedChecksum string
}

func (p *Parser) Add(b byte) {
	switch p.State {
	case STATE_OPEN:
		p.Sentence += string(b) //TODO: use bytes buffer
		p.checksum = p.checksum ^ b
		break
	case STATE_CHECKING:
		p.expectedChecksum += string(b)

		if len(p.expectedChecksum) == 2 {
			str := strconv.FormatUint(uint64(p.checksum), 16)
			if len(str) == 1 {
				str = "0" + str
			}

			if strings.ToUpper(str) == p.expectedChecksum {
				p.Output <- p.Sentence
				p.Count++
			} /* else {
				fmt.Println("ERROR (", p.Sentence, ")", p.expectedChecksum, "!=", str)
			}*/

			p.State = STATE_CLOSED
		}
		break
	case STATE_CLOSED:
		break
	default:
		break
	}
}

func (p *Parser) Push(b byte) {
	switch b {
	case 36: //$
		p.State = STATE_OPEN

		p.Sentence = ""
		p.checksum = 0
		p.expectedChecksum = ""
		break
	case 42: //*
		if p.State == STATE_OPEN {
			p.State = STATE_CHECKING
			p.expectedChecksum = ""
		}
		break
	default:
		p.Add(b)
		break
	}
}

func (parser *Parser) Parse(input *bytes.Buffer) (uint64, error) {
	return parser.ParseInto(input, parser.Output)
}

func (parser *Parser) ParseInto(input *bytes.Buffer, out chan string) (uint64, error) {
	parser.State = STATE_CLOSED
	parser.Output = out //TODO (FIXME)

	b, err := input.ReadByte()
	for err == nil {
		parser.Push(b)
		b, err = input.ReadByte()
	}

	parser.State = STATE_CLOSED //reset state

	if err != io.EOF { //EOF is ok, all other errors are not
		return parser.Count, err
	} else {
		return parser.Count, nil
	}
}

func Checksum(sentence string) string {
	reader := strings.NewReader(sentence)
	var cs byte
	var b byte
	for i := 0; i < len(sentence); i++ {
		b, _ = reader.ReadByte()
		cs = cs ^ b
	}

	str := strconv.FormatUint(uint64(cs), 16)
	if len(str) == 1 {
		str = "0" + str
	}
	return str
}

//------------------------------------------------------------
//sentence

func NewSentenceFromString(sentence string) *Sentence {
	s := new(Sentence)
	s.FromString(sentence)
	return s
}

type Sentence struct {
	Kind  string
	Data  []string
	Count int

	Valid bool
}

func (s *Sentence) FromString(sentence string) {
	s.Data = strings.Split(sentence, ",")
	s.Count = len(s.Data)
	s.Valid = false

	if s.Count > 0 {
		h := s.Data[0]
		l := len(h)
		if l == 5 {
			s.Kind = h[2:]
			s.Valid = true
		}
	}
}

//----------------------------------------------------------
// Builder

type Builder struct {
	Input  chan string
	Output chan *Sentence
	Quit   chan int
}

func (b *Builder) Process() {
	for message := range b.Input {
		s := NewSentenceFromString(message)
		b.Output <- s
	}
	close(b.Output)
}

//-------------------------------------------------------
// Pipeline

func NewPipeline() (p *Pipeline) {
	p = &Pipeline{}
	p.create()
	return p
}

type Pipeline struct {
	Parser  *Parser
	Builder *Builder

	Raw    chan string
	Output chan *Sentence
	Quit   chan int
}

func (p *Pipeline) create() {
	p.Raw = make(chan string, 100)
	p.Output = make(chan *Sentence, 100)

	p.Parser = &Parser{Output: p.Raw}

	p.Builder = &Builder{
		Input:  p.Raw,
		Output: p.Output,
	}

	go p.Builder.Process()
}

func (p *Pipeline) Push(data []byte) (uint64, error) {
	buf := bytes.NewBuffer(data)

	c, err := p.Parser.Parse(buf)

	return c, err
}

func (p *Pipeline) Close() {
	close(p.Raw)

}
