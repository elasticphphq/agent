package laravel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type QueueMetrics struct {
	Size int `json:"size"`
}

type QueueSizes map[string]map[string]QueueMetrics

func GetQueueSizes(appPath string, phpBinary string, queueMap map[string][]string) (*QueueSizes, error) {
	if len(queueMap) == 0 {
		return &QueueSizes{}, nil
	}

	script := `use Illuminate\Queue\QueueManager;
$manager = app(QueueManager::class);
$sizes = [];`

	for conn, queues := range queueMap {
		quoted := make([]string, len(queues))
		for i, q := range queues {
			quoted[i] = fmt.Sprintf(`'%s'`, q)
		}
		queueList := fmt.Sprintf("array(%s)", strings.Join(quoted, ", "))

		script += fmt.Sprintf(`
foreach (%s as $q) {
	try {
		$sizes["%s:" . $q] = $manager->connection("%s")->size($q);
	} catch (\Throwable $e) {
		$sizes["%s:" . $q] = -1;
	}
}`, queueList, conn, conn, conn)
	}

	script += `

echo json_encode($sizes);`

	cmd := exec.Command(phpBinary, "artisan", "tinker", "--execute", script)
	cmd.Dir = filepath.Clean(appPath)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("artisan tinker failed: %w\nOutput: %s", err, out.String())
	}

	var sizes map[string]int
	if err := json.Unmarshal(out.Bytes(), &sizes); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w\nOutput: %s", err, out.String())
	}

	result := QueueSizes{}
	for k, v := range sizes {
		parts := strings.SplitN(k, ":", 2)
		if len(parts) != 2 {
			continue
		}
		conn, queue := parts[0], parts[1]
		if _, ok := result[conn]; !ok {
			result[conn] = make(map[string]QueueMetrics)
		}
		result[conn][queue] = QueueMetrics{Size: v}
	}
	return &result, nil
}
