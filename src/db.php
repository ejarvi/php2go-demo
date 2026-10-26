<?php

declare(strict_types=1);

function db(): PDO
{
    static $pdo = null;
    if ($pdo === null) {
        // Some hosts give the database as a local socket, written "localhost:/path/to/mysql.sock".
        $host = config('DB_HOST');
        $socket = str_contains($host, ':/') ? substr($host, strpos($host, ':') + 1) : null;
        $dsn = $socket !== null
            ? sprintf('mysql:unix_socket=%s;dbname=%s;charset=utf8mb4', $socket, config('DB_NAME'))
            : sprintf('mysql:host=%s;port=%s;dbname=%s;charset=utf8mb4', $host, config('DB_PORT'), config('DB_NAME'));
        $pdo = new PDO($dsn, config('DB_USER'), config('DB_PASSWORD'), [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::ATTR_EMULATE_PREPARES => false,
        ]);
    }
    return $pdo;
}

function now(): DateTimeImmutable
{
    return new DateTimeImmutable('now');
}

function today(): DateTimeImmutable
{
    return now()->setTime(0, 0);
}

// The database stores times as local DATETIME values in the club's time zone.
function sql_time(DateTimeInterface $time): string
{
    return $time->format('Y-m-d H:i:s');
}

function query(string $sql, array $params = []): PDOStatement
{
    $statement = db()->prepare($sql);
    $statement->execute($params);
    return $statement;
}
