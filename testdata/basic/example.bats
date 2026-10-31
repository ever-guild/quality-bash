@test "default succeeds" {
    run bash "$BATS_TEST_DIRNAME/example.sh"
    [ "$status" -eq 0 ]
    [ "$output" = hello ]
}
