#!/usr/bin/env python3
import json
import math
import re
import sys
from collections import defaultdict


def percentile(values, percent):
    values = sorted(values)
    if not values:
        return 0
    position = (len(values) - 1) * percent / 100
    lower = math.floor(position)
    upper = math.ceil(position)
    if lower == upper:
        return values[lower]
    return values[lower] * (upper - position) + values[upper] * (position - lower)


def sample_group(request_id):
    if not request_id.startswith("prodperf-"):
        return ""
    match = re.fullmatch(r"(.+)-\d{8}-\d+", request_id[len("prodperf-"):])
    return match.group(1) if match else ""


def main(paths):
    groups = defaultdict(list)
    for path in paths:
        with open(path, encoding="utf-8", errors="replace") as source:
            for line in source:
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if event.get("content") != "http_request_completed":
                    continue
                group = sample_group(str(event.get("request_id", "")))
                if group:
                    groups[group].append(event)

    if not groups:
        raise SystemExit("未找到 request_id 以 prodperf- 开头的 http_request_completed 日志")

    print("group\tn\tstatus_2xx\terrors\tp50_ms\tp95_ms\tp99_ms\tavg_bytes\tmax_bytes\tavg_sql_count\tmax_sql_count\tavg_sql_ms\tmax_sql_ms\tslow_sql\tcache_hits\tcache_misses\tcache_hit_pct\tcache_fallbacks\tcache_decode_errors\tcache_redis_errors\tcache_write_errors")
    for group in sorted(groups):
        rows = groups[group]
        durations = [float(row.get("duration_ms", 0)) for row in rows]
        response_bytes = [float(row.get("response_bytes", 0)) for row in rows]
        sql_counts = [float(row.get("sql_count", 0)) for row in rows]
        sql_durations = [float(row.get("sql_duration_ms", 0)) for row in rows]
        statuses = [int(row.get("status", 0)) for row in rows]
        slow_sql = sum(int(row.get("slow_sql_count", 0)) for row in rows)
        cache_hits = sum(int(row.get("cache_hits", 0)) for row in rows)
        cache_misses = sum(int(row.get("cache_misses", 0)) for row in rows)
        cache_fallbacks = sum(int(row.get("cache_fallbacks", 0)) for row in rows)
        cache_decode_errors = sum(int(row.get("cache_decode_errors", 0)) for row in rows)
        cache_redis_errors = sum(int(row.get("cache_redis_errors", 0)) for row in rows)
        cache_write_errors = sum(int(row.get("cache_write_errors", 0)) for row in rows)
        cache_reads = cache_hits + cache_misses
        cache_hit_percent = cache_hits * 100 / cache_reads if cache_reads else 0
        success = sum(200 <= status < 300 for status in statuses)
        errors = len(rows) - success
        average = lambda values: sum(values) / len(values) if values else 0
        print(
            f"{group}\t{len(rows)}\t{success}\t{errors}\t"
            f"{percentile(durations, 50):.1f}\t{percentile(durations, 95):.1f}\t{percentile(durations, 99):.1f}\t"
            f"{average(response_bytes):.1f}\t{max(response_bytes):.0f}\t"
            f"{average(sql_counts):.2f}\t{max(sql_counts):.0f}\t"
            f"{average(sql_durations):.1f}\t{max(sql_durations):.0f}\t{slow_sql}\t"
            f"{cache_hits}\t{cache_misses}\t{cache_hit_percent:.1f}\t{cache_fallbacks}\t"
            f"{cache_decode_errors}\t{cache_redis_errors}\t{cache_write_errors}"
        )


if __name__ == "__main__":
    if len(sys.argv) < 2:
        raise SystemExit(f"用法: {sys.argv[0]} /path/to/access.log [/path/to/rotated-access.log ...]")
    main(sys.argv[1:])
