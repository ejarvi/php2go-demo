<h1>Delete <?= e($event['title']) ?>?</h1>
<p>
  This removes the event and its <?= e(people_label($event['people'])) ?> signed up.
  It can't be undone.
</p>
<form method="post" action="/admin/events/<?= $event['id'] ?>/delete" class="button-row">
  <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">
  <input type="hidden" name="confirm" value="yes">
  <button type="submit" class="button button-danger">Delete event</button>
  <a href="/admin/events/<?= $event['id'] ?>">Cancel</a>
</form>
