<?php
// lokalni helper - isti algoritam kao generateToken() u app/includes/utils.php

function token_at_seed($seed) {
    srand($seed);
    $alphabet = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_';
    $last = strlen($alphabet) - 1;
    $buf = array();
    for ($i = 0; $i < 32; $i++) {
        $buf[] = $alphabet[rand(0, $last)];
    }
    return implode('', $buf);
}

if ($argc < 3) {
    fwrite(STDERR, "usage: php generateTokens.php FROM_MS TO_MS\n");
    exit(1);
}

$from_ms = (int) $argv[1];
$to_ms   = (int) $argv[2];

while ($from_ms < $to_ms) {
    echo token_at_seed($from_ms) . PHP_EOL;
    $from_ms++;
}
