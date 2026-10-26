<?php
$row = function (array $event): void { ?>
  <li class="admin-event">
    <a href="/admin/events/<?= $event['id'] ?>"><?= e($event['title']) ?></a>
    <span class="admin-event-meta">
      <?= e(event_day($event)) ?>
      · <?= $event['published'] ? 'Published' : '<strong>Draft</strong>' ?>
      · <?= $event['capacity'] === null
          ? e(people_label($event['people']))
          : e($event['people'] . ' / ' . $event['capacity']) ?>
    </span>
  </li>
<?php };
?>
<div class="page-heading">
  <h1>Events</h1>
  <a class="button" href="/admin/events/new">New event</a>
</div>

<h2>Upcoming</h2>
<?php if ($upcoming === []): ?>
  <p>No upcoming events.</p>
<?php else: ?>
  <ul class="admin-events">
    <?php foreach ($upcoming as $event) { $row($event); } ?>
  </ul>
<?php endif ?>

<h2>Past</h2>
<?php if ($past === []): ?>
  <p>No past events.</p>
<?php else: ?>
  <ul class="admin-events">
    <?php foreach ($past as $event) { $row($event); } ?>
  </ul>
<?php endif ?>

<h2>Recent activity</h2>
<?php if ($activity === []): ?>
  <p>Nothing has changed yet.</p>
<?php else: ?>
  <ul class="activity">
    <?php foreach ($activity as $entry): ?>
      <li>
        <span class="activity-time"><?= e(short_time($entry['created_at'])) ?></span>
        <?= e($entry['action']) ?><?= $entry['event_title'] !== null ? ': ' . e($entry['event_title']) : '' ?>
      </li>
    <?php endforeach ?>
  </ul>
<?php endif ?>
