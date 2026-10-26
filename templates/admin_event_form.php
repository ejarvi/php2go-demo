<?php
$action = $event === null ? '/admin/events' : '/admin/events/' . $event['id'];
$total = array_sum(array_column($signups, 'people'));
?>
<p><a href="/admin">← All events</a></p>
<h1><?= $event === null ? 'New event' : e($event['title']) ?></h1>

<?php if ($saved): ?>
  <p class="notice notice-success" role="status">Saved.</p>
<?php endif ?>

<?php if ($event !== null && $event['capacity'] !== null && $event['people'] > $event['capacity']): ?>
  <p class="notice notice-warning"><?= e(people_label($event['people'])) ?> <?= $event['people'] === 1 ? 'is' : 'are' ?> already signed up.</p>
<?php endif ?>

<?php if ($event !== null && $event['published']): ?>
  <p><a href="/events/<?= $event['id'] ?>">View the public page</a></p>
<?php endif ?>

<?php if ($errors): ?>
  <p class="notice notice-error" role="alert">Please fix the highlighted fields.</p>
<?php endif ?>

<form method="post" action="<?= e($action) ?>" class="form" novalidate>
  <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">

  <div class="field">
    <label for="title">Title</label>
    <input id="title" name="title" type="text" maxlength="120" required
      value="<?= e($form['title']) ?>"<?= field_error_attrs($errors, 'title') ?>>
    <?= field_error($errors, 'title') ?>
  </div>

  <div class="field-row">
    <div class="field">
      <label for="date">Date</label>
      <input id="date" name="date" type="date" required
        value="<?= e($form['date']) ?>"<?= field_error_attrs($errors, 'date') ?>>
      <?= field_error($errors, 'date') ?>
    </div>
    <div class="field">
      <label for="start_time">Start time</label>
      <input id="start_time" name="start_time" type="time" required
        value="<?= e($form['start_time']) ?>"<?= field_error_attrs($errors, 'start_time') ?>>
      <?= field_error($errors, 'start_time') ?>
    </div>
    <div class="field">
      <label for="end_time">End time (optional)</label>
      <input id="end_time" name="end_time" type="time"
        value="<?= e($form['end_time']) ?>"<?= field_error_attrs($errors, 'end_time') ?>>
      <?= field_error($errors, 'end_time') ?>
    </div>
  </div>

  <div class="field">
    <label for="location">Location</label>
    <input id="location" name="location" type="text" maxlength="200" required
      value="<?= e($form['location']) ?>"<?= field_error_attrs($errors, 'location') ?>>
    <?= field_error($errors, 'location') ?>
  </div>

  <div class="field">
    <label for="description">Description</label>
    <textarea id="description" name="description" rows="6" maxlength="5000"<?= field_error_attrs($errors, 'description') ?>><?= e($form['description']) ?></textarea>
    <?= field_error($errors, 'description') ?>
  </div>

  <div class="field">
    <label for="capacity">Capacity (optional)</label>
    <input id="capacity" name="capacity" type="number" min="1" max="500" step="1" inputmode="numeric"
      value="<?= e($form['capacity']) ?>"<?= field_error_attrs($errors, 'capacity') ?>>
    <?= field_error($errors, 'capacity') ?>
  </div>

  <div class="field field-check">
    <input id="published" name="published" type="checkbox" value="1"<?= $form['published'] ? ' checked' : '' ?>>
    <label for="published">Published</label>
  </div>

  <button type="submit" class="button"><?= $event === null ? 'Create event' : 'Save changes' ?></button>
</form>

<?php if ($event !== null): ?>
  <section aria-labelledby="signups-heading">
    <h2 id="signups-heading">Sign-ups</h2>
    <?php if ($signups === []): ?>
      <p>Nobody has signed up yet.</p>
    <?php else: ?>
      <div class="table-scroll" role="region" aria-labelledby="signups-heading" tabindex="0">
        <table>
          <thead>
            <tr><th scope="col">Name</th><th scope="col">Email</th><th scope="col">People</th><th scope="col">Signed up</th><th scope="col"><span class="visually-hidden">Actions</span></th></tr>
          </thead>
          <tbody>
            <?php foreach ($signups as $signup): ?>
              <tr>
                <td><?= e($signup['name']) ?></td>
                <td><?= e($signup['email']) ?></td>
                <td><?= $signup['people'] ?></td>
                <td><?= e(short_time($signup['created_at'])) ?></td>
                <td>
                  <form method="post" action="/admin/signups/<?= $signup['id'] ?>/delete" class="inline-form">
                    <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">
                    <button type="submit" class="link-button">Remove<span class="visually-hidden"> <?= e($signup['name']) ?></span></button>
                  </form>
                </td>
              </tr>
            <?php endforeach ?>
          </tbody>
          <tfoot>
            <tr><th scope="row">Total</th><td></td><td><?= $total ?></td><td></td><td></td></tr>
          </tfoot>
        </table>
      </div>
      <p><a href="/admin/events/<?= $event['id'] ?>/signups.csv">Download CSV</a></p>
    <?php endif ?>
  </section>

  <section aria-labelledby="delete-heading">
    <h2 id="delete-heading">Delete this event</h2>
    <form method="post" action="/admin/events/<?= $event['id'] ?>/delete">
      <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">
      <button type="submit" class="button button-danger">Delete event…</button>
    </form>
  </section>
<?php endif ?>
