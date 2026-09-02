# jjui end-to-end tests

This directory contains the end-to-end tests for jjui.

The tests run jjui as a real process connected to a pseudo-terminal (PTY).
They send keyboard input through the PTY and inspect the rendered terminal
screen, exercising jjui in the same way as an interactive user.

The suite runs inside Docker, where each test creates an isolated temporary
`jj` repository. Tests interact with that repository through jjui and use `jj`
commands to verify any resulting changes.

## Running the tests

From this directory, run the full suite with:

```sh
task test
```

To run a specific test, pass its name or a regular expression:

```sh
task test -- Test_Preview_AutoPlacementFollowsTerminalSize
```

The first run may take a few minutes while Docker builds the test image.
