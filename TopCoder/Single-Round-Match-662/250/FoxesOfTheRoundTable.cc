#include <bits/stdc++.h>
using namespace std;

typedef long long ll;

typedef pair<int, int> pii;
#define A first
#define B second

class FoxesOfTheRoundTable {

public:

	static const int MAXN = 100;
	int N;
	pii V[MAXN];

	vector <int> minimalDifference(vector <int> h) {
		N = int(h.size());
		for(int i = 0; i < N; i++) {
			V[i] = pii(h[i], i);
		}
		sort(V, V + N);
		vector<int> res;
		for(int i = 0; i < N; i+=2) {
			res.push_back(V[i].B);
		}
		for(int i = N / 2 * 2 - 1; i > 0; i -= 2) {
			res.push_back(V[i].B);
		}
		return res;
	}
};

// vim:ft=cpp
