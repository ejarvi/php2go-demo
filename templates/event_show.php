<?php
$left = places_left($event);
$started = event_has_started($event);
$canSignUp = $event['published'] && !$started && !event_is_full($event);
?>
<article class="event">
  <?php if (!$event['published']): ?>
    <p class="notice notice-warning"><strong>Draft: not visible to the public</strong></p>
  <?php endif ?>

  <?php if ($confirmed !== null): ?>
    <div class="notice notice-success" role="status">
      <p class="notice-title">You're signed up for <?= e($event['title']) ?>, <?= e(people_label($confirmed)) ?>.</p>
      <p><?= e(event_day($event)) ?> · <?= e(event_times($event)) ?> · <?= e($event['location']) ?></p>
      <p>Save or print this page so you have the details. No email will be sent.</p>
    </div>
  <?php endif ?>

  <h1><?= e($event['title']) ?></h1>
  <p class="event-when"><?= e(event_day($event)) ?> · <?= e(event_times($event)) ?></p>
  <p class="event-where"><?= e($event['location']) ?></p>

  <?php if (event_has_ended($event)): ?>
    <p class="notice">This event has ended.</p>
  <?php elseif ($left !== null): ?>
    <p class="places"><?= $left === 0 ? '<strong class="badge">Full</strong>' : e(places_label($left)) ?></p>
  <?php endif ?>

  <?php if ($event['description'] !== ''): ?>
    <div class="description"><p><?= description_html($event['description']) ?></p></div>
  <?php endif ?>
</article>

<section class="signup" aria-labelledby="signup-heading">
  <h2 id="signup-heading">Sign up</h2>

  <?php if ($notice !== null): ?>
    <p class="notice notice-error" role="alert"><?= e($notice) ?></p>
  <?php endif ?>

  <?php if ($canSignUp): ?>
    <?php if ($errors): ?>
      <p class="notice notice-error" role="alert">Please fix the highlighted fields.</p>
    <?php endif ?>
    <form method="post" action="/events/<?= $event['id'] ?>/signup" class="form" novalidate>
      <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">
      <input type="hidden" name="timer" value="<?= e(form_timer()) ?>">

      <div class="field">
        <label for="name">Name</label>
        <input id="name" name="name" type="text" maxlength="100" autocomplete="name" required
          value="<?= e($form['name']) ?>"<?= field_error_attrs($errors, 'name') ?>>
        <?= field_error($errors, 'name') ?>
      </div>

      <div class="field">
        <label for="email">Email</label>
        <p class="hint" id="email-hint">Only for the organizers to reach you if the event changes.</p>
        <input id="email" name="email" type="email" maxlength="254" autocomplete="email" required
          value="<?= e($form['email']) ?>"<?= isset($errors['email'])
            ? ' aria-invalid="true" aria-describedby="email-hint email-error"'
            : ' aria-describedby="email-hint"' ?>>
        <?= field_error($errors, 'email') ?>
      </div>

      <div class="field">
        <label for="people">How many people, including you</label>
        <input id="people" name="people" type="number" min="1" max="6" step="1" inputmode="numeric" required
          value="<?= e($form['people']) ?>"<?= field_error_attrs($errors, 'people') ?>>
        <?= field_error($errors, 'people') ?>
      </div>

      <div class="trap">
        <label for="website">Leave this field empty</label>
        <input id="website" name="website" type="text" tabindex="-1" autocomplete="off">
      </div>

      <button type="submit" class="button">Sign up</button>
    </form>
  <?php elseif ($started): ?>
    <p>Sign-ups have closed because this event has started.</p>
  <?php elseif ($event['published']): ?>
    <p>This event is full, so sign-ups have closed.</p>
  <?php else: ?>
    <p>Sign-ups open once the event is published.</p>
  <?php endif ?>
</section>
