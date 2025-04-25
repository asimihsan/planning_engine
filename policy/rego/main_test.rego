package gate

test_allow_when_within_limit {
    # Setup test case
    input := {
        "pending_delta": 100,
        "max_pending_allowed": 500
    }
    
    # Evaluate the policy
    result := response with input as input
    
    # Assert the result
    result.allow == true
    count(result.deny_reasons) == 0
}

test_deny_when_exceeds_limit {
    # Setup test case
    input := {
        "pending_delta": 600,
        "max_pending_allowed": 500
    }
    
    # Evaluate the policy
    result := response with input as input
    
    # Assert the result
    result.allow == false
    result.deny_reasons[0] == "pending_delta exceeds allowed limit"
}