#! /usr/bin/env python

"""
========================
Javascript UCD Generator
========================

This script generates a ``ucd.js`` file from the Unicode Character Database
(UCD).

Currently works for Unicode 5.2.

"""

from __future__ import with_statement

data = {}
ranges = []
in_range = False

with open('UnicodeData.txt', 'rb') as ucd:
    for line in ucd:
        line = line.split(';')
        codepoint = int(line[0], 16)
        name, category = line[1:3]
        if in_range:
            ranges[-1].extend([codepoint, category])
            in_range = False
        if name.endswith('First>'):
            ranges.append([codepoint])
            in_range = True
        if category not in data:
            data[category] = []
        data[category].append(codepoint)

groupings = {
    "CasedLetter": "Ll | Lt | Lu",
    "Letter": "Ll | Lm | Lo | Lt | Lu",
    "Mark": "Mc | Me | Mn",
    "Number": "Nd | Nl | No",
    "Other": "Cc | Cf | Cn | Co | Cs",
    "Punctuation": "Pc | Pd | Pe | Pf | Pi | Po | Ps",
    "Symbol": "Sc | Sk | Sm | So",
    "Separator": "Zl | Zp | Zs"
    }

output = []; out = output.append
def trim(output):
    output[-1] = output[-1][:-2] + '\n'

categories = sorted(data.keys())
groups = sorted(groupings.keys())
last_category = len(categories) - 1

out("""
// Generated by running ./makeucd.py
//
// DO NOT EDIT

var version = "5.2.0",
    cat,
    catranges,
    catnames,
""")

for group in groups:
    out("    %s,\n" % group)

for idx, category in enumerate(categories):
    out("    %s = %i,\n" % (category, 1 << idx))

out("    Unknown = Cn = 0;\n\n")
out("catnames = {\n")
out('    0: "Unknown",\n')

for idx, category in enumerate(categories):
    out('    %s: "%s",\n' % (1 << idx, category))

trim(output)
out("};\n\n")
out("cat = {")

for idx, category in enumerate(categories):
    out("\n    // General Category: %s\n" % category)
    codepoints = sorted(data[category])
    for cp in codepoints:
        out("    %s: %s,\n" % (cp, category))

trim(output)
out("};\n\n")
out("catranges = [\n")

for catrange in ranges:
    out("    [%i, %i, %s],\n" % tuple(catrange))

trim(output)
out("];\n")

out("""
// Various grouped general categories.

""")

for group in groups:
    out("%s = %s;\n" % (group, groupings[group]))

out("""
if ((typeof process !== "undefined") && (process !== null)) {
    exports.version = version;
    exports.cat = cat;
    exports.catranges = catranges;
    exports.catnames = catnames;
    exports.Unknown = Unknown
""")

for category in categories:
    # print "    %s = ucd.%s," % (category, category)
    out("    exports.%s = %s;\n" % (category, category))

for group in groups:
    # print "    %s = ucd.%s," % (group, group)
    out("    exports.%s = %s;\n" % (group, group))

out("""}

""")

ucd_file = open('ucd.js', 'wb')
ucd_file.write(''.join(output))
ucd_file.close()
