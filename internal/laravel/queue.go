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
	Driver          *string  `json:"driver"`
	Size            *int     `json:"size"`
	Pending         *int     `json:"pending"`
	Scheduled       *int     `json:"scheduled"`
	Reserved        *int     `json:"reserved"`
	OldestPending   *int     `json:"oldest_pending"`
	Failed          *int     `json:"failed"`
	OldestFailed    *int     `json:"oldest_failed"`
	NewestFailed    *int     `json:"newest_failed"`
	Failed1Min      *int     `json:"failed_1m"`
	Failed5Min      *int     `json:"failed_5m"`
	Failed10Min     *int     `json:"failed_10m"`
	FailedRate1Min  *float32 `json:"failed_rate_1m"`
	FailedRate5Min  *float32 `json:"failed_rate_5m"`
	FailedRate10Min *float32 `json:"failed_rate_10m"`
	ParseError      any      `json:"error"`
}

type QueueSizes map[string]map[string]QueueMetrics

func GetQueueSizes(appPath string, phpBinary string, queueMap map[string][]string) (*QueueSizes, error) {
	if len(queueMap) == 0 {
		return &QueueSizes{}, nil
	}

	script := `use Illuminate\Queue\QueueManager;
use Illuminate\Queue\Failed\FailedJobProviderInterface;
use Carbon\Carbon;

$manager = app(QueueManager::class);
$failedJobsProvider = app(FailedJobProviderInterface::class);
$now = now();
$sizes = [];`

	for conn, queues := range queueMap {
		quoted := make([]string, len(queues))
		for i, q := range queues {
			quoted[i] = fmt.Sprintf(`'%[1]s'`, q)
		}
		queueList := fmt.Sprintf("[%[1]s]", strings.Join(quoted, ", "))

		script += fmt.Sprintf(`
foreach (%[2]s as $q) {
	try {
		$sizes["%[1]s"][$q] = ['size' => null, 'pending' => null, 'delayed' => null, 'oldest_pending' => null, 'failed' => null, 'failed_rate' => null, 'failed_avg_time' => null];
		$connection = $manager->connection("%[1]s");
		if ($connection instanceof Illuminate\Queue\DatabaseQueue) {
			$sizes["%[1]s"][$q]['driver'] = "database";
			try {
				$db = $connection->getDatabase();
				$reflection = new ReflectionClass($connection);
				$property = $reflection->getProperty('table');
				$property->setAccessible(true);
				$table = $property->getValue($connection);
				$oldestPending = $db->table($table)->where("queue", $q)->whereNull("reserved_at")->orderBy("created_at")->value("created_at");

				$sizes["%[1]s"][$q]['pending'] = $db->table($table)->where("queue", $q)->whereNull("reserved_at")->where("available_at", "<=", $now->timestamp)->count();
				$sizes["%[1]s"][$q]['scheduled'] = $db->table($table)->where("queue", $q)->where("available_at", ">", $now->timestamp)->count();
				$sizes["%[1]s"][$q]['reserved'] = $db->table($table)->where("queue", $q)->whereNotNull("reserved_at")->count();
				$sizes["%[1]s"][$q]['oldest_pending'] = $oldestPending ? (int) now()->diffInSeconds(Carbon::createFromTimestamp($oldestPending), true) : null;
			} catch (\Throwable $e) {
				$sizes["%[1]s"][$q]['error'] = $e->getMessage();
			}
		}
		if ($connection instanceof Illuminate\Queue\RedisQueue) {
			$sizes["%[1]s"][$q]['driver'] = "redis";
			try {
				$redis = $connection->getConnection();
				$queueKey = $connection->getQueue($q);
	
				$sizes["%[1]s"][$q]['size'] = $redis->llen($queueKey);
				$sizes["%[1]s"][$q]['pending'] = $redis->llen($queueKey);
				$sizes["%[1]s"][$q]['scheduled'] = $redis->zcard($queueKey.':delayed');
				$sizes["%[1]s"][$q]['reserved'] = $redis->zcard($queueKey.':reserved');
	
				$oldestRaw = $redis->lindex($queueKey, 0);
				if ($oldestRaw) {
					$decoded = json_decode($oldestRaw, true);
					if (isset($decoded['createdAt'])) {
						$sizes["%[1]s"][$q]['oldest_pending'] = $decoded['createdAt'] ? (int) Carbon::createFromTimestamp($decoded['createdAt'])->diffInSeconds($now, true) : null;
					}
				}
			} catch (\Throwable $e) {
				$sizes["%[1]s"][$q]['error'] = $e->getMessage();
			}
		}
		$sizes["%[1]s"][$q]['size'] = $manager->connection("%[1]s")->size($q);

		try {
			if ($failedJobsProvider instanceof Illuminate\Queue\Failed\DatabaseFailedJobProvider
				|| $failedJobsProvider instanceof Illuminate\Queue\Failed\DatabaseUuidFailedJobProvider) {
			
				$failedProviderReflection = new ReflectionClass($failedJobsProvider);
				$method = $failedProviderReflection->getMethod('getTable');
				$method->setAccessible(true);
				$query = $method->invoke($failedJobsProvider);

				$minutes = [1, 5, 10];
				$failed = [];
				$failedRates = [];
			
				$baseQuery = $query->where('connection', '%[1]s')->where('queue', $q);;
	
				foreach ($minutes as $min) {
					$from = now()->subMinutes($min);
			
					$count = (clone $baseQuery)
						->where('failed_at', '>=', $from)
						->count();
			
					$failed[$min] = $count;
					$failedRates[$min] = round($count / $min, 2);
				}
	
				$oldestFailed = (clone $baseQuery)
					->whereNotNull('failed_at')
					->orderBy('failed_at', 'asc')
					->value('failed_at');
			
				$newestFailed = (clone $baseQuery)
					->whereNotNull('failed_at')
					->orderBy('failed_at', 'desc')
					->value('failed_at');
			
				$sizes["%[1]s"][$q]['failed'] = (clone $baseQuery)->count() ?? null;
				$sizes["%[1]s"][$q]['failed_rate_1m'] = $failedRates[1] ?? null;
				$sizes["%[1]s"][$q]['failed_rate_5m'] = $failedRates[5] ?? null;
				$sizes["%[1]s"][$q]['failed_rate_10m'] = $failedRates[10] ?? null;
				$sizes["%[1]s"][$q]['failed_1m'] = $failed[1] ?? null;
				$sizes["%[1]s"][$q]['failed_5m'] = $failed[5] ?? null;
				$sizes["%[1]s"][$q]['failed_10m'] = $failed[10] ?? null;
				$sizes["%[1]s"][$q]['oldest_failed'] = $oldestFailed ? (int) Carbon::parse($oldestFailed)->diffInSeconds($now, true) : null;
				$sizes["%[1]s"][$q]['newest_failed'] = $newestFailed ? (int) Carbon::parse($newestFailed)->diffInSeconds($now, true) : null;
			
			} else {
				$sizes["%[1]s"][$q]['error'] = "Unknown class ". $failedJobsProvider;
			}
		} catch (\Throwable $e) {
			$sizes["%[1]s"][$q]['error'] = $e->getMessage();
		}
		
	} catch (\Throwable $e) {
		$sizes["%[1]s"][$q]['size'] = null;
		$sizes["%[1]s"][$q]['error'] = $e->getMessage();
	}
}`, conn, queueList)
	}

	script += `

echo json_encode($sizes);`

	cmd := exec.Command(phpBinary, "-d", "error_reporting=E_ALL & ~E_DEPRECATED", "artisan", "tinker", "--execute", script)
	cmd.Dir = filepath.Clean(appPath)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("artisan tinker failed: %w\nOutput: %s", err, out.String())
	}

	result := QueueSizes{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w\nOutput: %s", err, out.String())
	}

	return &result, nil
}
