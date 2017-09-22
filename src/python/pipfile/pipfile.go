package pipfile

type Lock struct {
	Meta struct {
		Requires struct {
			Version string `json:"python_version"`
		} `json:"requires"`
	} `json:"_meta"`
}
