<h1>Upcoming events</h1>

<?php if ($events === []): ?>
  <p>No events are planned right now. Check back soon.</p>
<?php else: ?>
  <ul class="event-list">
    <?php foreach ($events as $event): ?>
      <li class="event-card">
        <h2><a href="/events/<?= $event['id'] ?>"><?= e($event['title']) ?></a></h2>
        <p class="event-when"><?= e(event_day($event)) ?> · <?= e(event_times($event)) ?></p>
        <p class="event-where"><?= e($event['location']) ?></p>
        <?php if (event_is_full($event)): ?>
          <p><strong class="badge">Full</strong></p>
        <?php endif ?>
      </li>
    <?php endforeach ?>
  </ul>
<?php endif ?>
