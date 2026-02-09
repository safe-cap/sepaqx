status_counts = {}

response = function(status, headers, body)
  status_counts[status] = (status_counts[status] or 0) + 1
end

done = function(summary, latency, requests)
  local total = 0
  local ok = 0
  io.write("Status codes:\n")
  for code, count in pairs(status_counts) do
    io.write(string.format("  %s: %d\n", code, count))
    total = total + count
    if tonumber(code) == 200 then
      ok = count
    end
  end
  if total > 0 then
    io.write(string.format("  200 rate: %.2f%%\n", (ok * 100.0) / total))
    local duration_sec = summary.duration / 1000000
    if duration_sec > 0 then
      local okps = ok / duration_sec
      local errps = (total - ok) / duration_sec
      io.write(string.format("  ok/s: %.2f\n", okps))
      io.write(string.format("  err/s: %.2f\n", errps))
    end
  end
end
