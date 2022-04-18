package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	f, err := os.Open("test.json")
	if err != nil {
		log.Fatalf("open file: %s", err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	res, fail := unmarshalResponce(decoder)

	for {
		select {
		case item, openned := <-res.Results:
			if !openned {
				goto FINISH
			}

			log.Println("result", item.Field)
		case err, openned := <-fail:
			if !openned {
				goto FINISH
			}

			log.Fatalf("read results from file: %s", err)
		}
	}

FINISH:
	log.Println("total", res.Total)
}

type unmarshalResponseError struct {
	UnknownToken json.Token
}

func (e unmarshalResponseError) Error() string {
	return fmt.Sprintf("unknown %T: %v", e.UnknownToken, e.UnknownToken)
}

func unmarshalResponce(decoder *json.Decoder) (res *response, failed chan error) {
	_res := response{
		Results: make(resultsStream),
	}
	failed = make(chan error)

	go func() {
		defer close(_res.Results)
		defer close(failed)

		for {
			t, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				failed <- fmt.Errorf("decode next token: %w", err)
			}

			switch tv := t.(type) {
			case string:
				switch tv {
				case "results":
					if err = unmarshalResults(_res.Results, decoder); err != nil {
						err = fmt.Errorf("unmarshal results: %w", err)
					}
				case "total":
					res.Total, err = unmarshalResponseTotal(decoder)
					if err != nil {
						err = fmt.Errorf("unmarshal response total value: %w", err)
					}
				default:
					err = unmarshalResponseError{
						UnknownToken: t,
					}
				}
			default:
			}

			if err != nil {
				failed <- err
			}
		}
	}()

	return &_res, failed
}

func unmarshalResults(stream resultsStream, decoder *json.Decoder) (err error) {
	for {
		t, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode next token: %w", err)
		}

		switch tv := t.(type) {
		case json.Delim:
			switch tv {
			case '{':
				res, err := unmarshalResult(decoder)
				if err != nil {
					return fmt.Errorf("unmarshal result: %w", err)
				}

				stream <- res
			case '}', '[':
			case ']':
				return nil
			}
		default:
			return unmarshalResponseError{UnknownToken: t}
		}
	}
}

func unmarshalResult(decoder *json.Decoder) (res result, err error) {
	t, err := decoder.Token()
	if err != nil {
		return res, fmt.Errorf("decode field key next token: %w", err)
	}

	key, ok := t.(string)
	if !ok || key != "field" {
		return res, unmarshalResponseError{UnknownToken: t}
	}

	t, err = decoder.Token()
	if err != nil {
		return res, fmt.Errorf("decode field value next token: %w", err)
	}

	res.Field, ok = t.(string)
	if !ok {
		return res, unmarshalResponseError{UnknownToken: t}
	}

	return res, nil
}

func unmarshalResponseTotal(decoder *json.Decoder) (int, error) {
	t, err := decoder.Token()
	if err != nil {
		return 0, fmt.Errorf("decode total value in next token: %w", err)
	}

	switch tv := t.(type) {
	case float64:
		return int(tv), nil
	default:
		return 0, unmarshalResponseError{
			UnknownToken: t,
		}
	}
}

type response struct {
	Results resultsStream `json:"results"`
	Total   int           `json:"total"`
}

type resultsStream chan result

type result struct {
	Field string `json:"field"`
}
