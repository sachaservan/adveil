#include "wrapper.h"
#include <seal/seal.h>

using namespace std;
using namespace seal;

void* init_params(uint64_t num_items, uint64_t item_bytes, uint64_t poly_degree, uint64_t logt, uint64_t d) {

    EncryptionParameters *enc_params = new EncryptionParameters(scheme_type::BFV);
    PirParams *pir_params = new PirParams();
    gen_params(num_items, item_bytes, poly_degree, logt, d, *enc_params, *pir_params);
   
    struct Params *params = new Params(); 
    params->enc_params = enc_params;
    params->pir_params = pir_params;
    params->num_items = num_items;
    params->item_bytes = item_bytes;
    params->poly_degree = poly_degree;
    params->logt = logt;
    params->d = d;

    return params;
}

void* init_client_wrapper(void *params, uint64_t client_id) { 
    struct ClientWrapper *cw = new ClientWrapper();
    struct Params *p = (struct Params *)params;
    PIRClient *cli = new PIRClient(*(p->enc_params), *(p->pir_params));
    cw->client = cli;
    cw->params = p;
    cw->client_id = client_id;
    return cw;
}

void* gen_galois_keys(void *client_wrapper) {
    struct ClientWrapper *cw = (struct ClientWrapper *)client_wrapper;
    SerializedGaloisKeys *ser = new SerializedGaloisKeys();
    GaloisKeys galois_keys = cw->client->generate_galois_keys();
    string ser_keys = serialize_galoiskeys(galois_keys);
    char *str = new char [ser_keys.length()+1]; 
    memcpy(str, ser_keys.c_str(), sizeof(char) * (ser_keys.length()+1));
    ser->str = str;
    ser->str_len = ser_keys.length();
    ser->client_id = cw->client_id;
    return ser;
}

uint64_t fv_index(void *client_wrapper, uint64_t elem_index) {
    struct ClientWrapper *cw = (struct ClientWrapper *)client_wrapper;
    uint64_t size_per_item = cw->params->item_bytes;
    // index of FV plaintext
    uint64_t index = cw->client->get_fv_index(elem_index, size_per_item);

    return index; 
}

uint64_t fv_offset(void *client_wrapper, uint64_t elem_index) {
    struct ClientWrapper *cw = (struct ClientWrapper *)client_wrapper;
    uint64_t size_per_item = cw->params->item_bytes;
    // index of FV plaintext
    uint64_t index = cw->client->get_fv_index(elem_index, size_per_item);
    // offset in FV plaintext   
    uint64_t offset = cw->client->get_fv_offset(elem_index, size_per_item);

    return offset; 
}

void* gen_query(void *client_wrapper, uint64_t desiredIndex) {
    struct ClientWrapper *cw = (struct ClientWrapper *)client_wrapper;
    PirQuery query = cw->client->generate_query(desiredIndex);
    string query_ser = serialize_query(query);
    
    std::vector<seal::Ciphertext> size_test;
    size_test.push_back(query[0][0]);

    SerializedQuery *ser = new SerializedQuery();
    char *str = new char [query_ser.length()+1]; 
    memcpy(str, query_ser.c_str(), sizeof(char) * (query_ser.length()+1));
    ser->str = str;
    ser->str_len = query_ser.length();
    ser->ciphertext_size = serialize_ciphertexts(size_test).size();
    ser->count = 1;

    return ser;
}

char* recover(void *client_wrapper, void *serialized_answer) {
    struct ClientWrapper *cw = (struct ClientWrapper *)client_wrapper;
    struct SerializedAnswer *sa = (struct SerializedAnswer *)serialized_answer;

    string str(sa->str, sa->str_len);
    PirReply answer = deserialize_ciphertexts(sa->count, str, sa->ciphertext_size);
    Plaintext result = cw->client->decode_reply(answer);
    uint64_t size = ((cw->params->poly_degree * cw->params->logt) / 8);
    uint8_t* elems = new uint8_t[size]; 
    coeffs_to_bytes(cw->params->logt, result, elems, size);
    return (char*) elems; 
}


void* init_server_wrapper(void *params) {
    struct ServerWrapper *sw = new ServerWrapper();
    struct Params *p = (struct Params *)params;
    PIRServer *server = new PIRServer(*(p->enc_params), *(p->pir_params));
    sw->server = server;
    sw->params = p;
    return sw;
}

void set_galois_keys(void *server_wrapper, void *serialized_galois_keys) {
    struct ServerWrapper *sw = (ServerWrapper *)server_wrapper;
    struct SerializedGaloisKeys *k = (struct SerializedGaloisKeys *) serialized_galois_keys;
    string str(k->str, k->str_len);
    GaloisKeys *galois_keys = deserialize_galoiskeys(str);
    sw->server->set_galois_key(k->client_id, *galois_keys);
}

void setup_database(void *server_wrapper, char* data) {
    struct ServerWrapper *sw = (ServerWrapper *)server_wrapper;
    uint64_t size = sw->params->num_items * sw->params->item_bytes;
    auto db = make_unique<uint8_t[]>(size);
    memcpy(db.get(), data, size);
    sw->server->set_database(move(db), sw->params->num_items, sw->params->item_bytes);
    sw->server->preprocess_database();
}

void* gen_answer(void *server_wrapper, void *serialized_query) {
    struct ServerWrapper *sw = (ServerWrapper *)server_wrapper;
    struct SerializedQuery *sq = (SerializedQuery *)serialized_query;

    string serialized(sq->str, sq->str_len);
    PirQuery query = deserialize_query(
        sw->params->d, 
        sq->count, 
        serialized, 
        sq->ciphertext_size
    );

    PirReply res = sw->server->generate_reply(query, sq->client_id);
    string ser_ans = serialize_ciphertexts(res);

    std::vector<seal::Ciphertext> size_test;
    size_test.push_back(res[0]);

    SerializedAnswer *ans = new SerializedAnswer();

    char *str = new char [ser_ans.length()+1]; 
    memcpy(str, ser_ans.c_str(), sizeof(char) * (ser_ans.length()+1));
    ans->str = str;
    ans->str_len = ser_ans.length();
    ans->ciphertext_size = serialize_ciphertexts(size_test).size();
    ans->count = res.size();

    return ans;
}

extern void* gen_expanded_query(void *server_wrapper, void *serialized_query) {

    struct ServerWrapper *sw = (ServerWrapper *)server_wrapper;
    struct SerializedQuery *sq = (SerializedQuery *)serialized_query;

    string serialized(sq->str, sq->str_len);
    PirQuery query = deserialize_query(
        sw->params->d, 
        sq->count, 
        serialized, 
        sq->ciphertext_size
    );

    std::vector<std::vector<seal::Ciphertext>> res = sw->server->expand_query(query, sq->client_id);

    struct ExpandedQuery *eq = new ExpandedQuery();

    seal::Ciphertext *queries1 = new seal::Ciphertext[res[0].size()];
    memcpy(queries1, res[0].data(), sizeof(seal::Ciphertext)*res[0].size());
    eq->queries1 = queries1;
    eq->len1 = res[0].size();

    seal::Ciphertext *queries2 = new seal::Ciphertext[res[1].size()];
    memcpy(queries2, res[1].data(), sizeof(seal::Ciphertext)*res[1].size());
    eq->queries2 = queries2;
    eq->len2 = res[1].size();

    if (sw->params->d == 3) {
        seal::Ciphertext *queries3 = new seal::Ciphertext[res[2].size()];
        memcpy(queries3, res[2].data(), sizeof(seal::Ciphertext)*res[2].size());
        eq->queries3 = queries3;
        eq->len3 = res[2].size();
    }

    eq->client_id = sq->client_id;

    return eq;
}

extern void* gen_answer_with_expanded_query(void *server_wrapper, void *expanded_query) {

    struct ServerWrapper *sw = (ServerWrapper *)server_wrapper;
    struct ExpandedQuery *eq = (ExpandedQuery *)expanded_query;

    // TODO: this is incredibly hacky and potentially has memory leaks. 
    // The issue is that it's not trivial to convert between (void*)
    // and std::vector types. This was the best workaround found. 
    vector<vector<seal::Ciphertext>> queries(sw->params->d);
    seal::Ciphertext *q1 = new seal::Ciphertext[eq->len1];
    memcpy(q1, eq->queries1, sizeof(seal::Ciphertext) * eq->len1);
    vector<seal::Ciphertext> queries1(q1, q1 + eq->len1);

    seal::Ciphertext *q2 = new seal::Ciphertext[eq->len2];
    memcpy(q2, eq->queries2, sizeof(seal::Ciphertext) * eq->len2);
    vector<seal::Ciphertext> queries2(q2, q2 + eq->len2);

    queries[0] = queries1;
    queries[1] = queries2;

    if (sw->params->d == 3) {
        seal::Ciphertext *q3 = new seal::Ciphertext[eq->len2];
        memcpy(q3, eq->queries3, sizeof(seal::Ciphertext) * eq->len3);
        vector<seal::Ciphertext> queries3(q3, q3 + eq->len3);

        queries[2] = queries3;
    }

    PirReply res = sw->server->generate_reply_with_expanded_queries(queries, eq->client_id);
    string ser_ans = serialize_ciphertexts(res);

    std::vector<seal::Ciphertext> size_test;
    size_test.push_back(res[0]);

    SerializedAnswer *ans = new SerializedAnswer();

    char *str = new char [ser_ans.length()+1]; 
    memcpy(str, ser_ans.c_str(), sizeof(char) * (ser_ans.length()+1));
    ans->str = str;
    ans->str_len = ser_ans.length();
    ans->ciphertext_size = serialize_ciphertexts(size_test).size();
    ans->count = res.size();

    return ans;

}


void free_params(void *params) {
    struct Params *p = (struct Params *)params;
    free(p->pir_params);
    free(p->enc_params);
    free(p);
}

void free_client_wrapper(void *client_wrapper) {
    struct ClientWrapper *cw = (struct ClientWrapper *)client_wrapper;
    free(cw->client);
    free(cw);
}

void free_server_wrapper(void *server_wrapper) {
    struct ServerWrapper *sw = (ServerWrapper *)server_wrapper;
    free(sw->server);
    free(sw);
}

void free_expanded_query(void *expanded_query) {
    struct ExpandedQuery *eq = (ExpandedQuery *)expanded_query;
    free(eq->queries1);
    free(eq->queries2);
    if (eq->queries3 != NULL) {
        free(eq->queries3);
    }

    free(expanded_query);
}

void free_query(void *query) {
    struct SerializedQuery *q = (struct SerializedQuery *)query;
    free((char*)q->str);
    free(q);
}

void free_answer(void *answer) {
    struct SerializedAnswer *a = (struct SerializedAnswer *)answer;
    free((char*)a->str);
    free(a);
}