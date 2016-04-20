package nmea_test

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	. "golibs/gonmea"
	"testing"
	"time"
)

func Test_simple_parsing(t *testing.T) {
	b := []byte("$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47$GPGSA,A,3,04,05,,09,12*,,,24.1*39\r\nagafgsa$$$$$fgsfgafga$GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1*39")

	buf := bytes.NewBuffer(b)
	out := make(chan string, 100)

	p := new(Parser)
	c, err := p.ParseInto(buf, out)
	assert.Nil(t, err)

	for i := uint64(0); i < c; i++ {
		s := <-out
		t.Log(s)
	}
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

func Test_builder(t *testing.T) {
	in := make(chan string, 100)
	out := make(chan *Sentence, 100)
	quit := make(chan int)

	b := &Builder{
		Input:  in,
		Output: out,
		Quit:   quit,
	}

	go b.Process()

	in <- "GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,"
	in <- "GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1"

	time.Sleep(1 * time.Second)

	quit <- 1

	s := <-out
	assert.NotNil(t, s)
	assert.Equal(t, "GGA", s.Kind)

	s = <-out
	assert.NotNil(t, s)
	assert.Equal(t, "GSA", s.Kind)
}

func Test_pipeline(t *testing.T) {
	p := new(Pipeline)
	p.Create()

	b := []byte("$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47$GPGSA,A,3,04,05,,09,12*,,,24.1*39\r\nagafgsa$$$$$fgsfgafga$GPGSA,A,3,04,05,,09,12,,,24,,,,,2.5,1.3,2.1*39")
	p.Push(b)

	s := <-p.Output
	assert.NotNil(t, s)
	assert.Equal(t, "GGA", s.Kind)

	s = <-p.Output
	assert.NotNil(t, s)
	assert.Equal(t, "GSA", s.Kind)

	p.Close()
}
