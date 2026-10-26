<h1>Organizers</h1>

<form method="post" action="/signin" class="form form-narrow" novalidate>
  <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">
  <input type="hidden" name="next" value="<?= e($next) ?>">

  <?php if ($error !== null): ?>
    <p class="notice notice-error" role="alert" id="password-error"><?= e($error) ?></p>
  <?php endif ?>

  <div class="field">
    <label for="password">Password</label>
    <input id="password" name="password" type="password" autocomplete="current-password" required
      <?= $error !== null ? 'aria-invalid="true" aria-describedby="password-error"' : '' ?>>
  </div>

  <button type="submit" class="button">Sign in</button>
</form>
