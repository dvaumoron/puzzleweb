package errors

import "errors"

const QueryError = "?error="
const WrongLang = "wrong.lang"

var ErrorNotAuthorized = errors.New("error.not.authorized")
var ErrorTechnical = errors.New("error.technical.problem")
var ErrorUpdate = errors.New("error.update")
