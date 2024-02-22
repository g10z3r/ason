package ason

import (
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	textFileName = "testmain.go"
)

func TestGOARCH(t *testing.T) {
	assert.Equal(t, strconv.IntSize, _GOARCH())
}

func TestNewPosition(t *testing.T) {
	tokPos := token.Position{
		Filename: textFileName,
		Offset:   10,
		Line:     20,
		Column:   30,
	}

	pos := NewPosition(tokPos)
	assert.NotNil(t, pos)
	assert.Equal(t, tokPos.Filename, pos.Filename)
	assert.Equal(t, tokPos.Offset, pos.Offset())
	assert.Equal(t, tokPos.Line, pos.Line())
	assert.Equal(t, tokPos.Column, pos.Column())
}
