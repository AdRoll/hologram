# List of all return codes that we can return with an explanation.
# This way a client could figure out what happeend just by looking at our return code.

export ERRTEST=1    # Error found when running tests
export ERRCOMPILE=2 # Error found compiling hologram
export ERRLIN=3  # Error found building linux packages
export ERROSXPKG=4  # Error found building osx packages
export ERRDEPINST=5 # Error found when trying to install dependencies
export ERRARGS=6    # Invalid argument received by startup script
