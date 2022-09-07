import sys
import subprocess
import shlex
import csv
import json
import yaml
import jsonschema
import tempfile
import os


def test_destination_csv(cvTask: str):

    tmpCSV = tempfile.NamedTemporaryFile()

    subprocess.call(shlex.split(
        f'docker run -it --privileged --pid=host debian nsenter -t 1 -m -u -n -i sh -c "cat /var/lib/docker/volumes/airbyte/_data/connector-backend-test/_airbyte_raw_{cvTask}.csv"'),
        stdout=open(tmpCSV.name, "w"))

    # The VDP protocol YAML file downloaded during image build time
    with open("/usr/local/vdp/vdp_protocol.yaml") as f:
        jsonSchema = json.loads(json.dumps(yaml.safe_load(f)))

    # read csv file
    with open(tmpCSV.name, encoding="utf-8") as csvf:
        # load csv file data using csv library's dictionary reader
        csvReader = csv.DictReader(csvf)

        if len(list(csvReader)) == 0:
            sys.exit(1)

        # convert each csv row into python dict
        for row in csvReader:
            try:
                jsonschema.validate(
                    instance=json.loads(row["_airbyte_data"]),
                    schema=jsonSchema,
                    cls=jsonschema.Draft7Validator,
                )
            except:
                sys.exit(1)


if __name__ == "__main__":

    CV_TASKS = [
        "classification",
        "detection",
        "keypoint",
        "ocr",
        "unspecified"
    ]

    for cvTask in CV_TASKS:
        test_destination_csv(cvTask)

    sys.exit(0)
