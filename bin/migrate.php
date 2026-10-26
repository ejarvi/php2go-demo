<?php

declare(strict_types=1);

// Applies the migrations/*.sql files that haven't run yet, in name order. Safe to run
// more than once. The deploy runs it before the new release goes live (design/delivery.md).

if (PHP_SAPI !== 'cli') {
    exit(1);
}

require dirname(__DIR__) . '/src/bootstrap.php';

$pdo = db();

// One migration run at a time, even if two deploys or test servers start together.
if ((int) $pdo->query("SELECT GET_LOCK('php2go_migrate', 60)")->fetchColumn() !== 1) {
    fwrite(STDERR, "Another migration is still running.\n");
    exit(1);
}

$pdo->exec('CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(100) NOT NULL PRIMARY KEY,
    applied_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci');

$applied = $pdo->query('SELECT version FROM schema_migrations')->fetchAll(PDO::FETCH_COLUMN);

foreach (glob(dirname(__DIR__) . '/migrations/*.sql') as $file) {
    $version = basename($file, '.sql');
    if (in_array($version, $applied, true)) {
        continue;
    }
    // Comments are whole lines starting with --, and statements end with a semicolon at the end of a line.
    $script = preg_replace('/^\s*--.*$/m', '', (string) file_get_contents($file));
    foreach (preg_split('/;\s*$/m', $script) as $statement) {
        $sql = trim($statement);
        if ($sql !== '') {
            $pdo->exec($sql);
        }
    }
    query('INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)', [$version, sql_time(now())]);
    echo "Applied $version\n";
}

$pdo->query("SELECT RELEASE_LOCK('php2go_migrate')");
echo "Storage is up to date.\n";
