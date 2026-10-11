package projects

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRepositoryInventoryFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"hosts.ini", "hosts.yaml", "hosts", "site.yml", "README.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("test"), 0600))
	}
	req := httptest.NewRequest(http.MethodGet, "/inventory-files", nil)
	req = helpers.SetContextValue(req, "repository", db.Repository{GitURL: root})
	rec := httptest.NewRecorder()
	c := &RepositoryController{}
	c.GetRepositoryInventoryFiles(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var files []string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &files))
	assert.Equal(t, []string{"hosts", "hosts.ini", "hosts.yaml", "site.yml"}, files)

	// Sharing checkout logic must not change the existing playbook filter.
	rec = httptest.NewRecorder()
	c.GetRepositoryPlaybooks(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &files))
	assert.Equal(t, []string{"hosts.yaml", "site.yml"}, files)
}
