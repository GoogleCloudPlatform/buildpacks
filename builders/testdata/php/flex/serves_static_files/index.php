<?php

echo 'index.php';
// When redirected to front controller, make sure URL params are intact.
if (isset($_GET['name'])) {
    echo PHP_EOL;
    echo 'name is ' . $_GET['name'];
}
