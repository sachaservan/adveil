#include "wrapper.h"
#include <chrono>
#include <memory>
#include <random>
#include <cstdint>
#include <cstddef>

using namespace std::chrono;
using namespace std;
using namespace seal;

uint8_t* gen_random_db (uint64_t db_size, uint32_t num_bytes)
{
    uint8_t *db = new uint8_t[db_size * num_bytes];
    random_device rd;
    for (uint64_t i = 0; i < db_size; i++) {
        for (uint64_t j = 0; j < num_bytes; j++) {
            auto val = rd() % 256;
            db[(i * num_bytes) + j] = val;
        }
    }

  return db;
}

int main(int argc, char *argv[]) {

    // SEAL parameters 
    uint64_t num_items = 1 << 12;
    uint32_t item_bytes = 288; // in bytes (must be same as N for SPIR)
    uint32_t poly_degree = 2048;
    uint32_t logt = 12; 
    uint32_t d = 2;

    void *params = init_params(num_items, item_bytes, poly_degree, logt, d);
    void *cw = init_client_wrapper(params, 0);
    void *sw = init_server_wrapper(params);

    void *keys = gen_galois_keys(cw);
    set_galois_keys(sw, keys);

    uint8_t *db = gen_random_db(num_items, item_bytes);
    setup_database(sw, (char*) db);

    // pick random index 
    uint64_t elem_index = rand() % num_items;
    uint64_t index = fv_index(cw, elem_index);
    uint64_t offset = fv_offset(cw, elem_index);

    // query for this index 
    void *query = gen_query(cw, index);
    void *ans = gen_answer(sw, query);

    uint8_t *result = (uint8_t*) recover(cw, ans);

    // check that we retrieved the correct element
    for (uint32_t i = 0; i < item_bytes; i++) {
        if (result[(offset * item_bytes) + i] != db[(elem_index * item_bytes) + i]) {
            cout << "Main: elems " << (int)result[(offset * item_bytes) + i] << ", db "
                 << (int) db[(elem_index * item_bytes) + i] << endl;
            cout << "Main: PIR result wrong!" << endl;
            return -1;
        }
    }

    free_query(query);
    free_answer(ans);
    free_params(params);
    free_client_wrapper(cw);
    free_server_wrapper(sw);
    free(db);

    return 0;
}
