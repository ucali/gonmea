package nmea

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_simple_parsing(t *testing.T) {
	b := []byte("$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47$GPGSA,A,3,04,05,,09,12*,,,24.1*39\r\nagafgsa$$$$$fgsfgafga$GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1*39")

	buf := bytes.NewBuffer(b)
	out := make(chan string, 100)

	p := &parser{Output: out}
	c, err := p.Parse(buf)
	assert.Nil(t, err)

	for i := uint64(0); i < c; i++ {
		s := <-out
		t.Log(s)
	}
}

func Test_builder(t *testing.T) {
	in := make(chan string, 100)
	out := make(chan *Sentence, 100)

	b := &builder{
		Input:  in,
		Output: out,
	}

	go b.Process()

	in <- "GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,"
	in <- "GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1"

	time.Sleep(1 * time.Second)

	s := <-out
	assert.NotNil(t, s)
	assert.Equal(t, "GGA", s.Kind)

	s = <-out
	assert.NotNil(t, s)
	assert.Equal(t, "GSA", s.Kind)

	close(in)
}

func Test_message(t *testing.T) {
	s := &Sentence{}
	s.FromString("GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,")

	assert.True(t, s.Valid)
	assert.Equal(t, "GGA", s.Kind)

	s = &Sentence{}
	s.FromString("PGGA,123519,4807.03000,E,1,08,0.9,545.4,M,46.9,M,,")

	assert.False(t, s.Valid)

	s = NewSentenceFromString("GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1")
	assert.True(t, s.Valid)
	assert.Equal(t, "GSA", s.Kind)
}

func Test_pipeline(t *testing.T) {
	p := NewPipeline()

	b := []byte("çsd+è$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47$GPGSA,A,3,04,05,,09,12*,,,24.1*39\r\nagafgsa$$$$$fgsfgafga$GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1*39")
	p.Push(b)
	p.Push(b)
	p.Close()
	

	s := <-p.Output
	assert.NotNil(t, s)
	assert.Equal(t, "GGA", s.Kind)

	s = <-p.Output
	assert.NotNil(t, s)
	assert.Equal(t, "GSA", s.Kind)

}
