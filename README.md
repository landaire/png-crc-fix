# png-crc-fix

A tool for fixing the checksums in PNG chunks. The need for this tool came from
fuzzing image-related tools and needing to ensure that the checksums are valid
so that the image does not get rejected through basic checks.

# Installation

    go get gitlab.com/landaire/png-crc-fix

# Usage

    png-crc-fix FILE
