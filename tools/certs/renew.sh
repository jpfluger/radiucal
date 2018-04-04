#!/bin/bash
function print-bar()
{
    echo "
==================
$@
==================
Press <Enter> to continue"
    read
}
print-bar "this script assumes all passwords are the same, this is viable for a restricted area LAN at best. you will be prompted for a password (in all future cases - use the same one)."
if [ -f passwords.mk ]; then
    print-bar "WARN - a passwords.mk file already exists..."
fi
echo "removing previous certs"
rm -f *.pem *.der *.csr *.crt *.key *.p12 serial* index.txt*
echo -n Password:
read -s password
echo
echo -n Verify:
read -s verify
if [[ "$password" != "$verify" ]]; then
    echo
    echo "passwords did not match..."
    exit 1
fi
for f in $(ls *.cnf); do
    sed -i "s/input_password =[[:blank:]]*$/input_password = $password/g" $f
    sed -i "s/output_password =[[:blank:]]*$/output_password = $password/g" $f
done
echo "rebuilding certs"
./bootstrap
run=$?
if [ $run -ne 0 ]; then
    print-bar "ERROR - unable to process certificate renewal!!!"
else
    print-bar "make sure to update the /etc/hostapd/hostapd.conf with the private key password"
fi
