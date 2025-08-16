<?php

class StringUtils
{
    public static function injectRequestedAt(string $json, string $requestedAt): string
    {
        $insert = '"requestedAt":"' . $requestedAt . '",';

        return preg_replace('/^{/', '{' . $insert, $json, 1);
    }
}