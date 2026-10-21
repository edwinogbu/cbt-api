package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"
)

// BackupInfo describes one backup file found on disk - read-only,
// filesystem-sourced status for the admin dashboard (Master
// Implementation Plan section 7: "Display ... backup status"). This
// never performs a backup itself - scripts/backup-school-node.sh (run
// manually or via cron, see README-SCHOOL-NODE.md) is the only thing
// that writes to this directory, matching standard practice of running
// pg_dump as a trusted OS-level process rather than over HTTP (avoids
// request timeouts on a large database and avoids ever putting
// database credentials in reach of this HTTP handler).
type BackupInfo struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

// ListBackups reports what scripts/backup-school-node.sh has actually
// written to BACKUP_DIR (default ./backups, matching the script's own
// default) - never a claimed/assumed backup status. An admin/
// super_admin/platform_owner (the same role gate as every other
// /admin/* route) can see this to confirm backups are actually
// running, not just configured to.
func ListBackups(c *gin.Context) {
	dir := os.Getenv("BACKUP_DIR")
	if dir == "" {
		dir = "./backups"
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		// No backup has ever been taken yet - an empty, successful
		// response is the honest answer, not an error.
		c.JSON(http.StatusOK, gin.H{"backup_dir": dir, "backups": []BackupInfo{}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read backup directory"})
		return
	}

	backups := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only list files this node's own backup script actually
		// produces - a backup directory is a reasonable place for an
		// admin to also keep unrelated notes/scratch files, which
		// shouldn't be reported as if they were backups.
		if filepath.Ext(name) != ".gz" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Filename:  name,
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	sort.Slice(backups, func(i, j int) bool { return backups[i].CreatedAt > backups[j].CreatedAt })

	c.JSON(http.StatusOK, gin.H{"backup_dir": dir, "backups": backups})
}
