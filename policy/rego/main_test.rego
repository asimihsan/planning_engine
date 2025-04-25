package gate

test_allow_when_within_limit if {
    # Setup test case
    test_data := {
        "pending_delta": 100,
        "max_pending_allowed": 500
    }

    # Evaluate the policy
    result := response with input as test_data

    # Assert the result
    result.allow == true
    result.deny_reasons == []
}

test_deny_when_exceeds_limit if {
    # Setup test case
    test_data := {
        "pending_delta": 600,
        "max_pending_allowed": 500
    }

    # Evaluate the policy
    result := response with input as test_data

    # Assert the result
    result.allow == false
    result.deny_reasons[0] == "pending_delta exceeds allowed limit"
}
