#!/bin/bash

# Golem Advanced Integration Test Suite

set -e

echo "==========================================="
echo "   Running Golem Integration Test Suite"
echo "==========================================="

if [ ! -f "./golem" ]; then
    echo "Error: golem binary not found. Run 'go build -o golem ./cmd/golem'"
    exit 1
fi

ITERATIONS=${1:-1}

for RUN in $(seq 1 $ITERATIONS); do
    echo -e "\n=== Iteration $RUN of $ITERATIONS ==="

    # 1. Setup fresh fixtures
    ./demo/setup_fixtures.sh

    # --- Category A: Safe Internal Operations ---

    echo -e "\n[TEST A1] Standard Safe Copy"
    yes y | ./golem "copy the files named README.md from demo/fixtures/projects into a new folder called demo/fixtures/projects-dump/docs" > /dev/null
    if [ -f "demo/fixtures/projects-dump/docs/golem-core_README.md" ]; then
        echo "✅ A1 passed!"
    else
        echo "❌ A1 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST A2] Flattened File Move"
    yes y | ./golem "move all text files from demo/fixtures/projects into demo/fixtures/projects-dump/flat" > /dev/null
    if [ -f "demo/fixtures/projects-dump/flat/golem-core_docs_api.txt" ]; then
        echo "✅ A2 passed!"
    else
        echo "❌ A2 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST A3] Wildcard Processing"
    yes y | ./golem "copy all *.go files from demo/fixtures/projects to demo/fixtures/projects-dump/go-files" > /dev/null
    if [ -f "demo/fixtures/projects-dump/go-files/golem-core_main.go" ]; then
        echo "✅ A3 passed!"
    else
        echo "❌ A3 failed on run $RUN!"
        exit 1
    fi

    # --- Category B: Sandbox Escapes & External Locations ---

    echo -e "\n[TEST B1] Destination Sandbox Escape"
    rm -rf /tmp/golem-test-escape
    yes y | ./golem "copy all .js files from demo/fixtures/projects into /tmp/golem-test-escape" > /dev/null
    if [ -f "/tmp/golem-test-escape/web-frontend_src_app.js" ]; then
        echo "✅ B1 passed!"
    else
        echo "❌ B1 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST B2] Source Sandbox Escape"
    echo "External Source File" > ~/golem_test_env/deep-source/external_file.txt
    yes y | ./golem "copy the file external_file.txt from $HOME/golem_test_env/deep-source into the directory demo/fixtures/projects-dump/external" > /dev/null
    if [ -f "demo/fixtures/projects-dump/external/external_file.txt" ]; then
        echo "✅ B2 passed!"
    else
        echo "❌ B2 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST B3] Total OS Operation (Both Escape)"
    rm -rf /tmp/golem-test-total-escape
    mkdir -p /tmp/golem-test-total-escape
    echo "Total Escape File" > /tmp/golem-test-total-escape/total.txt
    yes y | ./golem "move the file total.txt from /tmp/golem-test-total-escape into the directory $HOME/golem_test_env/total-dest" > /dev/null
    if [ -f "$HOME/golem_test_env/total-dest/total.txt" ]; then
        echo "✅ B3 passed!"
    else
        echo "❌ B3 failed on run $RUN!"
        exit 1
    fi

    # --- Category C: Path Intelligence & LLM Hallucinations ---

    echo -e "\n[TEST C1] Fuzzy Source Matching"
    yes y | ./golem "copy the file README.md from demo/fixtures/peojects into the directory demo/fixtures/projects-dump/fuzzy" > /dev/null
    if [ -f "demo/fixtures/projects-dump/fuzzy/golem-core_README.md" ]; then
        echo "✅ C1 passed!"
    else
        echo "❌ C1 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST C2] Intelligent Source Deep Search"
    yes y | ./golem "copy the file deep_file.txt from deep-source into the directory demo/fixtures/projects-dump/deep-source-test" > /dev/null
    if [ -f "demo/fixtures/projects-dump/deep-source-test/deep_file.txt" ]; then
        echo "✅ C2 passed!"
    else
        echo "❌ C2 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST C3] Destination Missing Parent Search"
    yes y | ./golem "copy the file golem-core_main.go from demo/fixtures/projects-dump/go-files into the directory deep-dest-parent/new-folder" > /dev/null
    if [ -f "$HOME/golem_test_env/deep-dest-parent/new-folder/golem-core_main.go" ]; then
        echo "✅ C3 passed!"
    else
        echo "❌ C3 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST C4] Hallucinated Relative Dot"
    yes y | ./golem "copy the file deep_file.txt from ./golem_test_env/deep-source into the directory demo/fixtures/projects-dump/dot-test" > /dev/null
    if [ -f "demo/fixtures/projects-dump/dot-test/deep_file.txt" ]; then
        echo "✅ C4 passed!"
    else
        echo "❌ C4 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST C5] Hallucinated File in Source"
    yes y | ./golem "copy the file demo/fixtures/projects/golem-core/LICENSE.md to the directory demo/fixtures/projects-dump/file-in-source" > /dev/null
    if [ -f "demo/fixtures/projects-dump/file-in-source/golem-core_LICENSE.md" ] || [ -f "demo/fixtures/projects-dump/file-in-source/LICENSE.md" ]; then
        echo "✅ C5 passed!"
    else
        echo "❌ C5 failed on run $RUN!"
        exit 1
    fi

    # --- Category D: Destructive Operations & Blocking ---

    echo -e "\n[TEST D1] Whitelisted Deletion"
    yes y | ./golem "delete the build folder in demo/fixtures/projects" > /dev/null
    if [ ! -d "demo/fixtures/projects/build" ]; then
        echo "✅ D1 passed!"
    else
        echo "❌ D1 failed on run $RUN!"
        exit 1
    fi

    echo -e "\n[TEST D2] Blocked Deletion Attempt"
    yes y | ./golem "delete the directory demo/fixtures/projects/web-frontend/src" > /dev/null
    if [ -d "demo/fixtures/projects/web-frontend/src" ]; then
        echo "✅ D2 passed!"
    else
        echo "❌ D2 failed on run $RUN! Protected directory was deleted!"
        exit 1
    fi

    # --- Category E: Collision Handling (R/C/S) ---

    echo -e "\n[TEST E1] Collision: Skip (s) Move - Should preserve source"
    mkdir -p demo/fixtures/collisions/src demo/fixtures/collisions/dst
    echo "Source File" > demo/fixtures/collisions/src/conflict.txt
    echo "Dest File" > demo/fixtures/collisions/dst/conflict.txt
    yes "s" | ./golem --auto-confirm "move all text files from demo/fixtures/collisions/src into demo/fixtures/collisions/dst" > /dev/null
    if [ -f "demo/fixtures/collisions/src/conflict.txt" ] && [ -f "demo/fixtures/collisions/dst/conflict.txt" ]; then
        echo "✅ E1 passed!"
    else
        echo "❌ E1 failed on run $RUN! Source or Dest file missing after skip!"
        exit 1
    fi

    echo -e "\n[TEST E2] Collision: Create Copies (c) - Should rename with _2"
    yes "c" | ./golem --auto-confirm "copy conflict.txt from demo/fixtures/collisions/src into demo/fixtures/collisions/dst" > /dev/null
    if [ -f "demo/fixtures/collisions/dst/conflict_2.txt" ]; then
        echo "✅ E2 passed!"
    else
        echo "❌ E2 failed on run $RUN! Copy _2 was not created!"
        exit 1
    fi

    echo -e "\n[TEST E3] Collision: Replace (r) - Should overwrite destination"
    echo "New Source File" > demo/fixtures/collisions/src/conflict.txt
    yes "r" | ./golem --auto-confirm "move conflict.txt from demo/fixtures/collisions/src into demo/fixtures/collisions/dst" > /dev/null
    if [ ! -f "demo/fixtures/collisions/src/conflict.txt" ] && [ "$(cat demo/fixtures/collisions/dst/conflict.txt)" == "New Source File" ]; then
        echo "✅ E3 passed!"
    else
        echo "❌ E3 failed on run $RUN! Replace did not overwrite correctly!"
        exit 1
    fi

    # Clean up
    rm -rf /tmp/golem-test-escape
    rm -rf /tmp/golem-test-total-escape
done

echo -e "\n🎉 All advanced integration tests passed successfully across $ITERATIONS iterations!"
