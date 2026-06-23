package runtime

// ponytail: guardFinalAnswer removed — dead code, only guardFinalAnswerWithTrace used.

// shouldBufferFinalAnswer always returns true; all domains now use bufferFinal.
// ponytail: removed unused route parameter.
func shouldBufferFinalAnswer() bool {
	return true
}
