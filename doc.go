/*
	Package lookout provides gRPC APIs to implement analysis servers and
	clients.

	Services

	There are two services:

	1. Analyzer: analyzers process source code changes and provide analysis
	   results as text comments, possibly linked to specific parts of the code.

	2. Data: provides simple access to source code changes.
*/
package lookout
