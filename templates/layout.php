<?php
// Every page shares this layout (design/pages.md). An error page may be rendered when
// storage is down, so the organizer check here never lets that stop the page.
try {
    $organizer = is_organizer();
} catch (Throwable) {
    $organizer = false;
}
?>
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title><?= e($title) ?> · <?= e(config('CLUB_NAME')) ?></title>
<link rel="stylesheet" href="/assets/site.css?v=<?= e(rawurlencode(config('RELEASE'))) ?>">
</head>
<body>
<a class="skip-link" href="#main">Skip to content</a>
<header class="site-header">
  <div class="wrap header-inner">
    <a class="brand" href="/"><?= e(config('CLUB_NAME')) ?></a>
    <nav aria-label="Main">
      <ul class="nav">
        <li><a href="/">Events</a></li>
        <?php if ($organizer): ?>
          <li><a href="/admin">Admin</a></li>
          <li>
            <form method="post" action="/signout" class="inline-form">
              <input type="hidden" name="csrf" value="<?= e(csrf_token()) ?>">
              <button type="submit" class="link-button">Sign out</button>
            </form>
          </li>
        <?php endif ?>
      </ul>
    </nav>
  </div>
</header>
<main id="main" class="wrap">
<?= $content ?>
</main>
<footer class="site-footer">
  <div class="wrap footer-inner">
    <a href="/signin">Organizers</a>
    <span>Release <?= e(config('RELEASE')) ?></span>
  </div>
</footer>
</body>
</html>
