#!/bin/bash

# goiotop CLI Usage Examples
# This script demonstrates common usage patterns for goiotop

# Set the goiotop binary path (adjust as needed)
GOIOTOP="goiotop"

echo "=== goiotop CLI Usage Examples ==="
echo

# 1. Basic display (similar to iotop)
echo "1. Basic real-time display:"
echo "   $GOIOTOP display"
echo

# 2. Display with custom refresh rate
echo "2. Display with 5-second refresh:"
echo "   $GOIOTOP display --refresh 5"
echo

# 3. Display top 10 processes by disk I/O
echo "3. Top 10 processes by disk I/O:"
echo "   $GOIOTOP display --top 10 --sort disk-read"
echo

# 4. Filter by specific processes
echo "4. Monitor specific processes:"
echo "   $GOIOTOP display --processes postgres,mysql,redis"
echo

# 5. Single metric collection
echo "5. Collect metrics once:"
echo "   $GOIOTOP collect"
echo

# 6. Continuous collection
echo "6. Continuous collection every 2 seconds:"
echo "   $GOIOTOP collect --continuous --interval 2"
echo

# 7. Collect for specific duration
echo "7. Collect metrics for 60 seconds:"
echo "   $GOIOTOP collect --duration 60"
echo

# 8. Collect only disk metrics
echo "8. Collect only disk I/O metrics:"
echo "   $GOIOTOP collect --disk-only"
echo

# 9. Export to JSON file
echo "9. Export metrics to JSON:"
echo "   $GOIOTOP export --format json --destination metrics.json"
echo

# 10. Export to CSV with time range
echo "10. Export to CSV with time range:"
echo '    $GOIOTOP export --format csv --destination metrics.csv \'
echo '      --start "2024-01-01T00:00:00Z" --end "2024-01-02T00:00:00Z"'
echo

# 11. Export to HTTP endpoint
echo "11. Export to HTTP endpoint:"
echo '    $GOIOTOP export --format json \'
echo '      --destination "http://metrics.example.com/api/metrics" \'
echo '      --http-method POST'
echo

# 12. System information
echo "12. Display system information:"
echo "   $GOIOTOP system info"
echo

# 13. Detailed system info
echo "13. Detailed system information:"
echo "   $GOIOTOP system info --detailed"
echo

# 14. System health check
echo "14. Check system health:"
echo "   $GOIOTOP system health"
echo

# 15. Display collector capabilities
echo "15. Show collector capabilities:"
echo "   $GOIOTOP system capabilities"
echo

# 16. JSON output format
echo "16. Output in JSON format:"
echo "   $GOIOTOP display --format json"
echo

# 17. CSV output format
echo "17. Output in CSV format:"
echo "   $GOIOTOP collect --format csv --output metrics.csv"
echo

# 18. Batch mode (non-interactive)
echo "18. Run in batch mode:"
echo "   $GOIOTOP display --batch --format json"
echo

# 19. Custom configuration file
echo "19. Use custom configuration:"
echo "   $GOIOTOP --config /path/to/config.yaml display"
echo

# 20. Debug logging
echo "20. Enable debug logging:"
echo "   $GOIOTOP --log-level debug collect"
echo

echo
echo "=== Advanced Usage Examples ==="
echo

# Pipeline example 1: Continuous monitoring with filtering
echo "21. Monitor high I/O processes continuously:"
cat << 'EOF'
    $GOIOTOP display --refresh 1 \
      --sort total --top 10 --threshold 1000000
EOF
echo

# Pipeline example 2: Export and analyze
echo "22. Export metrics and analyze with jq:"
cat << 'EOF'
    $GOIOTOP collect --format json | \
      jq '.processMetrics | sort_by(.diskReadRate + .diskWriteRate) | reverse | .[0:5]'
EOF
echo

# Pipeline example 3: Monitor specific processes and log
echo "23. Monitor and log specific processes:"
cat << 'EOF'
    $GOIOTOP collect --continuous --interval 5 \
      --processes nginx,apache --format json \
      --output /var/log/webserver-io.json
EOF
echo

# Pipeline example 4: System health monitoring
echo "24. Automated health check with alerting:"
cat << 'EOF'
    if ! $GOIOTOP system health --format json | jq -e '.status == "healthy"' > /dev/null; then
      echo "System unhealthy! Sending alert..."
      # Send alert via email/slack/etc
    fi
EOF
echo

# Pipeline example 5: Batch export for analysis
echo "25. Batch export for time-series analysis:"
cat << 'EOF'
    # Collect metrics for 1 hour and export
    $GOIOTOP collect --duration 3600 --interval 10 \
      --format json --output hourly-metrics.json

    # Process with analysis tools
    python analyze_metrics.py hourly-metrics.json
EOF
echo

echo
echo "=== Automation Examples ==="
echo

# Cron job example
echo "26. Cron job for daily metrics collection:"
cat << 'EOF'
    # Add to crontab:
    0 0 * * * /usr/local/bin/goiotop collect --duration 86400 \
      --interval 60 --format json \
      --output /var/log/goiotop/metrics-$(date +\%Y\%m\%d).json
EOF
echo

# Systemd service example
echo "27. Systemd service for continuous monitoring:"
cat << 'EOF'
    # /etc/systemd/system/goiotop-monitor.service
    [Unit]
    Description=GoIOTop Continuous Monitoring
    After=network.target

    [Service]
    Type=simple
    ExecStart=/usr/local/bin/goiotop collect --continuous \
      --interval 5 --format json --output /var/log/goiotop/realtime.json
    Restart=always
    User=monitor

    [Install]
    WantedBy=multi-user.target
EOF
echo

# Docker example
echo "28. Monitor Docker containers:"
cat << 'EOF'
    # Monitor processes in Docker containers
    docker run --pid=host --network=host -v /proc:/proc:ro \
      goiotop:latest display
EOF
echo

# Kubernetes example
echo "29. Kubernetes DaemonSet for node monitoring:"
cat << 'EOF'
    # Deploy as DaemonSet to monitor all nodes
    kubectl apply -f - <<YAML
    apiVersion: apps/v1
    kind: DaemonSet
    metadata:
      name: goiotop-monitor
    spec:
      selector:
        matchLabels:
          app: goiotop
      template:
        metadata:
          labels:
            app: goiotop
        spec:
          hostPID: true
          hostNetwork: true
          containers:
          - name: goiotop
            image: goiotop:latest
            command: ["goiotop", "collect", "--continuous"]
            volumeMounts:
            - name: proc
              mountPath: /host/proc
              readOnly: true
          volumes:
          - name: proc
            hostPath:
              path: /proc
    YAML
EOF
echo

# Performance testing example
echo "30. Performance testing with goiotop:"
cat << 'EOF'
    # Start monitoring before load test
    $GOIOTOP collect --continuous --interval 1 \
      --output load-test-metrics.json &
    MONITOR_PID=$!

    # Run load test
    ./run-load-test.sh

    # Stop monitoring
    kill $MONITOR_PID

    # Analyze results
    $GOIOTOP export --format csv \
      --destination load-test-report.csv
EOF
echo

echo
echo "=== Tips and Best Practices ==="
echo
echo "• Use --batch mode when running in scripts or CI/CD pipelines"
echo "• Set appropriate --interval based on monitoring needs (lower = more overhead)"
echo "• Use --threshold to filter out low-activity processes"
echo "• Export to JSON for programmatic analysis"
echo "• Use CSV format for spreadsheet analysis"
echo "• Configure log levels appropriately (info for production, debug for troubleshooting)"
echo "• Use configuration files for consistent settings across environments"
echo "• Monitor system health regularly with automated checks"
echo "• Aggregate metrics over time for trend analysis"
echo "• Use process filters to focus on specific applications"
echo
echo "For version information: goiotop version"
echo "For more information, see: goiotop --help"