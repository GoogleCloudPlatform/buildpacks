<?php
$gotIniPath = php_ini_loaded_file();
$wantIniPath = '/workspace/php.ini';

if (!$gotIniPath) {
  echo 'FAIL. php.ini file is not loaded';
  return;
}

if ($gotIniPath != $wantIniPath) {
  echo 'FAIL. Unexpected php.ini path: got ' . $gotIniPath . ', want ' . $wantIniPath;
  return;
}

echo 'PASS_PHP_INI';