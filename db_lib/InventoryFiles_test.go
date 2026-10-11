package db_lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindInventoryFiles(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{
		"inventories/dev/hosts.ini", "inventories/prod/hosts.yaml", "inventories/stage/hosts.yml",
		"inventories/legacy/hosts", "tests/HOSTS.INI", "hosts.YAML",
		"inventories/dev/group_vars/all.yml", "inventories/prod/host_vars/web.yml",
		".git/config", ".venv/hosts.ini", "README.md", "deploy.py", "ansible.cfg",
	} {
		writeFile(t, root, file)
	}

	files, err := FindInventoryFiles(root)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"hosts.YAML", "inventories/dev/hosts.ini", "inventories/legacy/hosts",
		"inventories/prod/hosts.yaml", "inventories/stage/hosts.yml", "tests/HOSTS.INI",
	}, files)
}

func TestFindInventoryFilesLimitAndEmpty(t *testing.T) {
	root := t.TempDir()
	files, err := FindInventoryFiles(root)
	require.NoError(t, err)
	assert.Equal(t, []string{}, files)

	for i := 0; i < maxRepositoryFiles+5; i++ {
		writeFile(t, root, fmt.Sprintf("inventories/%03d.ini", i))
	}
	files, err = FindInventoryFiles(root)
	require.NoError(t, err)
	require.Len(t, files, maxRepositoryFiles)
	assert.Equal(t, "inventories/000.ini", files[0])
	assert.Equal(t, "inventories/029.ini", files[len(files)-1])
}
