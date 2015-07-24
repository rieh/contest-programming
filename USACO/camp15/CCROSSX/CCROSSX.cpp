#include<bits/stdc++.h>
using namespace std;

typedef long long ll;

const ll INF = 1e18;

struct node {
	node* p;
	node* c[2];
	inline bool d() { return this == p->c[1]; }

	ll ds; // delta slope
	ll dx; // delta position = length of segment
	ll s; // slope
	ll x; // position

	void update() {
		assert(ds > 0);
		assert(dx > 0);
		s = ds + (c[0] ? c[0]->s : 0) + (c[1] ? c[1]->s : 0);
		x = dx + (c[0] ? c[0]->x : 0) + (c[1] ? c[1]->x : 0);
	}

	void rot() {
		assert(p);

		bool v = d();
		node* pa = p;
		node* ch = c[!v];

		if(pa->p) pa->p->c[p->d()] = this;
		this->p = pa->p;

		pa->c[v] = ch;
		if(ch) ch->p = pa;
		
		this->c[!v] = pa;
		pa->p = this;

		pa->update();
	}

	node* splay(node* r = NULL) {
		while(p != r) {
			if(p->p != r) {
				if(p->d() == d()) {
					p->rot();
				} else {
					this->rot();
				}
			}
			rot();
		}
		update();
		return this;
	}

	// add slope to the node where left of tree = len
	// returns node to which slope was added
	node* add_slope(ll len, ll slo) {

		cerr << slo << ' ' << len << '\n';
		ll l = (c[0] ? c[0]->x : 0);
		ll r = (c[1] ? c[1]->x : 0);
		ll v = dx;
		assert(l + r + v == this->x);
		assert(len < this->x);
		assert(len >= 0);

		if(len < l) {
			assert(c[0]);
			return c[0]->add_slope(len, slo);
		} else if(len == l) {
			this->ds += slo;
			return this;
		} else if(len < l + v) {
			len -= l;

			node* n = new node;

			n->dx = v - len;
			n->ds = slo;

			this->dx = len;


			// insert n on right
			if(!c[1]) {
				c[1] = n;
				n->p = this;
			} else {
				cerr << "hi\n";
				n->p = c[1];
				while(n->p->c[0]) n->p = n->p->c[0];
				n->p->c[0] = n;
			}

			return n;
		} else {
			assert(len >= l + v);
			assert(c[1]);
			return c[1]->add_slope(len - l - v, slo);
		}
	}

	// add distance to the node where left equal of tree = slo
	// returns node to which distance was added
	node* add_dist(ll slo, ll len) {
		ll l = (c[0] ? c[0]->s : 0);
		ll r = (c[1] ? c[1]->s : 0);
		ll v = ds;
		assert(l + r + v == this->s);
		assert(slo <= this->s);
		assert(slo > 0);

		if(slo <= l) {
			return c[0]->add_dist(slo, len);
		} else if(slo == l + v) {
			this->dx += len;
			return this;
		} else if(slo < l + v) {
			slo -= l;

			node* n = new node;

			n->ds = slo;
			n->dx = len;

			this->ds = v - slo;

			if(!c[0]) {
				c[0] = n;
				n->p = this;
			} else {
				n->p = c[0];
				while(n->p->c[1]) n->p = n->p->c[1];
				n->p->c[1] = n;
			}

			return n;
		} else {
			assert(slo > l + v);
			assert(c[1]);
			return c[1]->add_dist(slo - l - v, len);
		}
	}

};

void dfs(node* n, ll &S, ll &V) {
	if(!n) return;
	dfs(n->c[0], S, V);
	S += n->ds;
	assert(S < 0);
	V += n->dx * S;
	dfs(n->c[1], S, V);
}

void print_tree(node* n) {
	if(!n) return;
	print_tree(n->c[0]);
	cerr << n->ds << ' ' << n->dx << '\n';
	print_tree(n->c[1]);
}

int main() {
	ll T; cin >> T;
	for(ll t = 0; t < T; t++) {
		ll N;
		ll D;
		cin >> N >> D;

		if(!D) {
			ll res = 0;
			ll H; cin >> H;
			for(int i = 1; i < N - 1; i++) {
				ll v; cin >> v;
				res += abs(v - H);
			}
			ll v; cin >> v;
			if(v != H) {
				cout << "impossible" << '\n';
			} else {
				cout << res << '\n';
			}
			continue;
		}

		assert(D > 0);

		ll H; cin >> H;

		ll V0 = 0; // value at start
		ll P0 = H - D; // position at start
		ll PF = H + D; // position at end
		ll S0 = -1; // slope at start
		ll SF = INF; // slope at end

		node* R = new node;
		R->ds = 1; // we do this just so that all slope deltas are positive
		R->dx = D + D;
		R->c[1] = new node;
		R->c[1]->ds = INF;
		R->c[1]->dx = INF;
		R->c[1]->update();
		R->update();

		cerr << S0 << ' ' << P0 << ' ' << PF << ' ' << '\n';
		print_tree(R);
		cerr << '\n';

		for(ll i = 1; i < N - 1; i++) {
			cin >> H;
			V0 += abs(H - P0);

			S0 --;
			SF --;
			if(H <= P0) {
				S0 += 2;
				SF += 2;
			} else if(H < PF) {
				R = R->add_slope(H - P0, 2)->splay();
				SF += 2;
			} else {
				// slope happens at end
			}

			cerr << S0 << ' ' << P0 << ' ' << PF << ' ' << '\n';
			print_tree(R);
			cerr << '\n';

			assert(S0 < 0 && 0 <= SF);

			R = R->add_dist(0 - S0, D + D)->splay();
			P0 -= D;
			PF += D;

			cerr << S0 << ' ' << P0 << ' ' << PF << ' ' << '\n';
			print_tree(R);
			cerr << '\n';
		}

		cin >> H;

		if(P0 <= H && H <= PF) {
			R = R->add_dist(1, 0 - S0)->splay();
			ll S = S0, V = 0;
			dfs(R, S, V);
			cout << V << '\n';
		} else {
			cout << "impossible" << '\n';
		}
	}

	return 0;
}
