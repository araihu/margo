package engines

import (
	"context"

	"github.com/araihu/margo/pdf"
)

func (discovery Discovery) Select() (pdf.Engine, Candidate, error) {
	for index, candidate := range discovery.candidates {
		if !candidate.Available || index >= len(discovery.engines) || discovery.engines[index] == nil {
			continue
		}
		return selectedEngine{engine: discovery.engines[index], candidate: candidate}, candidate, nil
	}
	return nil, Candidate{}, engineError("pdf.engine_unavailable", "no requested PDF engine is available", discovery.candidates)
}

type selectedEngine struct {
	engine    pdf.Engine
	candidate Candidate
}

func (selected selectedEngine) Name() string {
	return selected.candidate.Name
}

func (selected selectedEngine) Version(ctx context.Context) (string, error) {
	if selected.candidate.Version != "" {
		return selected.candidate.Version, nil
	}
	return selected.engine.Version(ctx)
}

func (selected selectedEngine) Export(ctx context.Context, request pdf.Request) (pdf.Result, error) {
	result, err := selected.engine.Export(ctx, request)
	if err != nil {
		return pdf.Result{}, err
	}
	result.Engine = pdf.EngineInfo{
		Name:    selected.candidate.Name,
		Version: selected.candidate.Version,
		Path:    selected.candidate.Path,
		Source:  string(selected.candidate.Source),
	}
	return result.Clone(), nil
}
