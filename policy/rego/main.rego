package gate

default allow := false
default deny_reasons := []

# Simple policy for initial testing: check if pending_delta is within allowed limit
allow if {
    input.pending_delta <= input.max_pending_allowed
}

deny_reasons := ["pending_delta exceeds allowed limit"] if {
    not allow
    input.pending_delta > input.max_pending_allowed
}

# Return a structured response for easier consumption by the engine
response := {
    "allow": allow,
    "deny_reasons": deny_reasons
} if true
