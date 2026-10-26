-- The records in design/data.md. Times are in the club's time zone (D-006);
-- the site always passes them in, and never relies on the database's clock.

CREATE TABLE events (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    title VARCHAR(120) NOT NULL,
    description TEXT NOT NULL,
    starts_at DATETIME NOT NULL,
    ends_at DATETIME NULL,
    location VARCHAR(200) NOT NULL,
    capacity SMALLINT UNSIGNED NULL,
    published TINYINT(1) NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    KEY events_starts_at (starts_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Email comparison is case-insensitive through the collation, so signing up
-- twice with different capitalization still finds the earlier sign-up.
CREATE TABLE signups (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    event_id INT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(254) NOT NULL,
    people TINYINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY signups_event_email (event_id, email),
    CONSTRAINT signups_event FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE sessions (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    code_hash CHAR(64) NOT NULL,
    expires_at DATETIME NOT NULL,
    password_fingerprint CHAR(64) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY sessions_code_hash (code_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- No foreign key: an entry outlives the event it describes, so it keeps the title.
CREATE TABLE activity (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    action VARCHAR(50) NOT NULL,
    event_id INT UNSIGNED NULL,
    event_title VARCHAR(120) NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    KEY activity_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
