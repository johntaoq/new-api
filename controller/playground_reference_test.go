package controller

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePlaygroundReferenceImage(t *testing.T) {
	png := append([]byte{}, []byte("\x89PNG\r\n\x1a\n")...)
	png = append(png, bytes.Repeat([]byte{0}, 32)...)

	contentType, err := validatePlaygroundReferenceImage(png, "application/octet-stream")
	require.NoError(t, err)
	require.Equal(t, "image/png", contentType)
}

func TestValidatePlaygroundReferenceImageRejectsInvalidContent(t *testing.T) {
	_, err := validatePlaygroundReferenceImage([]byte("not an image"), "text/plain")
	require.ErrorContains(t, err, "not an image")

	_, err = validatePlaygroundReferenceImage(nil, "image/png")
	require.ErrorContains(t, err, "empty")
}

func TestValidatePlaygroundReferenceImageRejectsOversize(t *testing.T) {
	_, err := validatePlaygroundReferenceImage(
		make([]byte, playgroundReferenceImageMaxBytes+1),
		"image/png",
	)
	require.ErrorContains(t, err, "exceeds 25 MB")
}
