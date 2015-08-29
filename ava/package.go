package main

type responseFn func(string) string

type AvaPackage struct {
	Responses map[string]responseFn
}

func (ap *AvaPackage) RespondsTo(inputs []string, fn responseFn) {
	for _, in := range inputs {
		ap.Responses[in] = fn
	}
}
