<?php

declare(strict_types=1);

// Deletes what the design says not to keep (design/data.md). Shared hosting has no
// scheduler we can set up from the repository (D-014), so requests run this now and then.
function housekeeping(): void
{
    $now = now();
    query(
        'DELETE s FROM signups s JOIN events e ON e.id = s.event_id WHERE COALESCE(e.ends_at, e.starts_at) < ?',
        [sql_time($now->modify('-30 days'))],
    );
    query('DELETE FROM sessions WHERE expires_at < ?', [sql_time($now)]);
    query('DELETE FROM activity WHERE created_at < ?', [sql_time($now->modify('-1 year'))]);
}
