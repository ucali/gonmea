package nmea

import (
	"bytes"
	"io"
	"strconv"
	"strings"
)

const (
	stateOpen     = 1
	stateClosed   = 2
	stateChecking = 3
)

//-------------------------------------------------------
// Pipeline

func NewPipeline() (p *Pipeline) {
	p = &Pipeline{}
	p.create()
	return p
}

type Pipeline struct {
	parser  *parser
	builder *builder

	Raw    chan string
	Output chan *Sentence
}

func (p *Pipeline) create() {
	p.Raw = make(chan string, 100)
	p.Output = make(chan *Sentence, 100)

	p.parser = &parser{Output: p.Raw}

	p.builder = &builder{
		Input:  p.Raw,
		Output: p.Output,
	}

	go p.builder.Process()
}

func (p *Pipeline) Push(data []byte) (uint64, error) {
	buf := bytes.NewBuffer(data)

	return p.parser.Parse(buf)
}

func (p *Pipeline) Close() {
	close(p.Raw)

}

// parser

type parser struct {
	State    int
	Sentence bytes.Buffer
	Count    uint64
	Output   chan string

	checksum         byte
	expectedChecksum bytes.Buffer
}

func (p *parser) Add(b byte) {
	switch p.State {
	case stateOpen:
		p.Sentence.WriteByte(b) //TODO: use bytes buffer
		p.checksum = p.checksum ^ b
	case stateChecking:
		p.expectedChecksum.WriteByte(b)

		if p.expectedChecksum.Len() == 2 {
			str := strconv.FormatUint(uint64(p.checksum), 16)
			if len(str) == 1 {
				str = "0" + str
			}

			if strings.ToUpper(str) == p.expectedChecksum.String() {
				p.Output <- p.Sentence.String()
				p.Count++
			}

			p.State = stateClosed
		}
	case stateClosed:
		break
	}
}

func (p *parser) Push(b byte) {
	switch b {
	case 36: //$
		p.State = stateOpen
		p.checksum = 0

		p.Sentence.Reset()
		p.expectedChecksum.Reset()
	case 42: //*
		if p.State == stateOpen {
			p.State = stateChecking
		}
	default:
		p.Add(b)
	}
}

func (p *parser) Parse(input *bytes.Buffer) (uint64, error) {
	p.State = stateClosed

	b, err := input.ReadByte()
	for err == nil {
		p.Push(b)
		b, err = input.ReadByte()
	}

	p.State = stateClosed //reset state

	if err == io.EOF { //EOF is ok, all other errors are not
		return p.Count, nil
	}
	return p.Count, err
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
// builder

type builder struct {
	Input  chan string
	Output chan *Sentence
}

func (b *builder) Process() {
	for message := range b.Input {
		s := NewSentenceFromString(message)
		b.Output <- s
	}
	close(b.Output)
}
