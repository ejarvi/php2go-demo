<?php

declare(strict_types=1);

// The only thing the migrate key can run. authorized_keys forces every connection with
// that key to run this script, whatever command was sent (D-015). It accepts one command,
// "apply <release>": it applies that release's storage changes, then removes all but
// the five newest releases. The deploy copies it to the top of the app folder.

if (PHP_SAPI !== 'cli') {
    exit(1);
}

$command = getenv('SSH_ORIGINAL_COMMAND') ?: '';
if (!preg_match('/^apply (r[0-9]+\+[0-9a-f]{7,40})$/', $command, $m)) {
    fwrite(STDERR, "migrate-gate: expected \"apply <release>\"\n");
    exit(2);
}

$releases = __DIR__ . '/releases';
$release = "$releases/{$m[1]}";
if (!is_file("$release/bin/migrate.php")) {
    fwrite(STDERR, "migrate-gate: release {$m[1]} hasn't been uploaded\n");
    exit(1);
}

// migrate.php exits with an error itself if anything goes wrong.
require "$release/bin/migrate.php";

// Old releases are only kept for a look back; rolling back is a revert and a deploy (D-016).
$all = glob("$releases/r*", GLOB_ONLYDIR) ?: [];
usort($all, fn (string $a, string $b): int => filemtime($b) <=> filemtime($a));
foreach (array_slice($all, 5) as $old) {
    if ($old !== $release) {
        remove_tree($old);
    }
}

function remove_tree(string $path): void
{
    $items = new RecursiveIteratorIterator(
        new RecursiveDirectoryIterator($path, FilesystemIterator::SKIP_DOTS),
        RecursiveIteratorIterator::CHILD_FIRST,
    );
    foreach ($items as $item) {
        $item->isDir() && !$item->isLink() ? rmdir($item->getPathname()) : unlink($item->getPathname());
    }
    rmdir($path);
}
