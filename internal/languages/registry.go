package languages

import (
	"errors"
	"sync"
)

var (
	ErrLanguageNotFound = errors.New("language not found")
)

type Registry struct {
	mu        sync.RWMutex
	languages map[string]Language
}

func NewRegistry() *Registry {
	r := &Registry{
		languages: make(map[string]Language),
	}
	r.registerDefaults()
	return r
}

func (r *Registry) Register(lang Language) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.languages[lang.ID] = lang
}

func (r *Registry) Get(id string) (Language, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lang, ok := r.languages[id]
	if !ok {
		return Language{}, ErrLanguageNotFound
	}
	return lang, nil
}

func (r *Registry) List() []Language {
	r.mu.RLock()
	defer r.mu.RUnlock()
	langs := make([]Language, 0, len(r.languages))
	for _, l := range r.languages {
		langs = append(langs, l)
	}
	return langs
}

func (r *Registry) registerDefaults() {
	r.Register(Language{
		ID:   "cpp",
		Name: "C++",
		Config: RuntimeConfig{
			Image:          "gcc:13",
			SourceFile:     "solution.cpp",
			CompileCommand: []string{"g++", "solution.cpp", "-O2", "-o", "solution"},
			RunCommand:     []string{"./solution"},
		},
	})

	r.Register(Language{
		ID:   "python",
		Name: "Python",
		Config: RuntimeConfig{
			Image:      "python:3.11-slim",
			SourceFile: "solution.py",
			RunCommand: []string{"python", "solution.py"},
		},
	})

	r.Register(Language{
		ID:   "javascript",
		Name: "JavaScript",
		Config: RuntimeConfig{
			Image:      "node:20-slim",
			SourceFile: "solution.js",
			RunCommand: []string{"node", "solution.js"},
		},
	})

	r.Register(Language{
		ID:   "typescript",
		Name: "Typescript",
		Config: RuntimeConfig{
			Image:          "node:20-slim",
			SourceFile:     "solution.ts",
			CompileCommand: []string{"tsc", "solution.ts"},
			RunCommand:     []string{"node", "solution.js"},
		},
	})
}
