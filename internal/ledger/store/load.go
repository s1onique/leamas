package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// readJSONFile reads and strictly decodes a single JSON record file.
func readJSONFile(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return nil
}

// listJSONFiles returns the sorted .json file names in dir.
func listJSONFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func loadEpics(dir string) ([]domain.Epic, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.Epic
	for _, name := range names {
		var e domain.Epic
		if err := readJSONFile(filepath.Join(dir, name), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EpicID < out[j].EpicID })
	return out, nil
}

func loadItems(dir string) ([]domain.WorkItem, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.WorkItem
	for _, name := range names {
		var it domain.WorkItem
		if err := readJSONFile(filepath.Join(dir, name), &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	domain.SortItems(out)
	return out, nil
}

func loadActs(dir string) ([]domain.ActExecution, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.ActExecution
	for _, name := range names {
		var a domain.ActExecution
		if err := readJSONFile(filepath.Join(dir, name), &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	domain.SortActs(out)
	return out, nil
}

func loadEvidence(dir string) ([]domain.EvidenceRefRecord, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.EvidenceRefRecord
	for _, name := range names {
		var e domain.EvidenceRefRecord
		if err := readJSONFile(filepath.Join(dir, name), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	domain.SortEvidence(out)
	return out, nil
}

func loadDecisions(dir string) ([]domain.Decision, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.Decision
	for _, name := range names {
		var d domain.Decision
		if err := readJSONFile(filepath.Join(dir, name), &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DecisionID < out[j].DecisionID })
	return out, nil
}

func loadKnowledge(dir string) ([]domain.Knowledge, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.Knowledge
	for _, name := range names {
		var k domain.Knowledge
		if err := readJSONFile(filepath.Join(dir, name), &k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].KnowledgeID < out[j].KnowledgeID })
	return out, nil
}

func loadMigrations(dir string) ([]domain.MigrationRecord, error) {
	names, err := listJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []domain.MigrationRecord
	for _, name := range names {
		var rec domain.MigrationRecord
		if err := readJSONFile(filepath.Join(dir, name), &rec); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MigrationID < out[j].MigrationID })
	return out, nil
}
