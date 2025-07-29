<?php

class DateTimeUtils
{
    public static function strToTimeWithMicro(string $dateTime): int
    {
        $secs = (string) strtotime($dateTime);

        $dateTimeImmutable = new DateTimeImmutable($dateTime);
        $microseconds = $dateTimeImmutable->format("u");

        return (int) ($secs.$microseconds);
    }
}