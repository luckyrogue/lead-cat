package microsoft

import "golang.org/x/oauth2"

type savingSource struct {
	base oauth2.TokenSource
	last string
	save func(*oauth2.Token)
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		s.last = tok.AccessToken
		if s.save != nil {
			s.save(tok)
		}
	}
	return tok, nil
}
