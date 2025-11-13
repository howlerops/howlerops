#!/bin/bash

# Fix s.eventLogger.LogSecurityEvent (multiline) in auth/two_factor.go
sed -i.bak '115s/s\.eventLogger\.LogSecurityEvent(/_ = s.eventLogger.LogSecurityEvent(/' internal/auth/two_factor.go
sed -i.bak '151s/s\.eventLogger\.LogSecurityEvent(/_ = s.eventLogger.LogSecurityEvent(/' internal/auth/two_factor.go
sed -i.bak '192s/s\.eventLogger\.LogSecurityEvent(/_ = s.eventLogger.LogSecurityEvent(/' internal/auth/two_factor.go
sed -i.bak '202s/s\.eventLogger\.LogSecurityEvent(/_ = s.eventLogger.LogSecurityEvent(/' internal/auth/two_factor.go
sed -i.bak '219s/s\.eventLogger\.LogSecurityEvent(/_ = s.eventLogger.LogSecurityEvent(/' internal/auth/two_factor.go
sed -i.bak '249s/s\.eventLogger\.LogSecurityEvent(/_ = s.eventLogger.LogSecurityEvent(/' internal/auth/two_factor.go

# Fix m.eventLogger.LogSecurityEvent in middleware/ip_whitelist.go
sed -i.bak '92s/m\.eventLogger\.LogSecurityEvent(/_ = m.eventLogger.LogSecurityEvent(/' internal/middleware/ip_whitelist.go

# Fix mock service calls in email/email_test.go  
sed -i.bak '204s/mockSvc\.SendVerificationEmail(/_ = mockSvc.SendVerificationEmail(/' internal/email/email_test.go
sed -i.bak '214s/mockSvc\.SendVerificationEmail(/_ = mockSvc.SendVerificationEmail(/' internal/email/email_test.go
sed -i.bak '215s/mockSvc\.SendPasswordResetEmail(/_ = mockSvc.SendPasswordResetEmail(/' internal/email/email_test.go
sed -i.bak '216s/mockSvc\.SendWelcomeEmail(/_ = mockSvc.SendWelcomeEmail(/' internal/email/email_test.go
sed -i.bak '285s/mockSvc\.SendVerificationEmail(/_ = mockSvc.SendVerificationEmail(/' internal/email/email_test.go

# Fix es.Disconnect and os.Disconnect in examples
sed -i.bak '37s/es\.Disconnect()/_ = es.Disconnect() \/\/ Best-effort disconnect/' pkg/database/examples/elasticsearch_example.go
sed -i.bak '152s/os\.Disconnect()/_ = os.Disconnect() \/\/ Best-effort disconnect/' pkg/database/examples/elasticsearch_example.go

# Fix conn.Close in pool.go
sed -i.bak '306s/conn\.Close()/_ = conn.Close() \/\/ Best-effort close/' pkg/storage/turso/pool.go

# Clean up backup files
find . -name "*.go.bak" -delete

echo "All remaining fixes applied!"
