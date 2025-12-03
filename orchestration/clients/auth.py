"""Authentication utilities for JWT token generation and validation"""
from datetime import datetime, timedelta
from typing import List, Optional
import jwt
import os
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.backends import default_backend


class JWTHelper:
    """Helper class for JWT token generation and validation"""

    def __init__(self, keys_dir: Optional[str] = None):
        """
        Initialize JWT helper.

        Args:
            keys_dir: Directory containing private_key.pem and public_key.pem.
                     If None, uses DEMO_KEYS_DIR env var or generates new keys.
        """
        if keys_dir is None:
            keys_dir = os.getenv("DEMO_KEYS_DIR", "/tmp/demo-keys")

        private_key_path = os.path.join(keys_dir, "private_key.pem")
        public_key_path = os.path.join(keys_dir, "public_key.pem")

        # Try to load existing keys from MCP server
        if os.path.exists(private_key_path) and os.path.exists(public_key_path):
            try:
                # Load private key
                with open(private_key_path, 'rb') as f:
                    self.private_key = serialization.load_pem_private_key(
                        f.read(),
                        password=None,
                        backend=default_backend()
                    )

                # Load public key
                with open(public_key_path, 'rb') as f:
                    self.public_key = serialization.load_pem_public_key(
                        f.read(),
                        backend=default_backend()
                    )

                print(f"✓ Loaded RSA keys from {keys_dir}")
            except Exception as e:
                print(f"⚠️ Failed to load keys from {keys_dir}: {e}")
                print("⚠️ Generating new keys (tokens will not work with MCP server)")
                self._generate_keys()
        else:
            print(f"⚠️ Keys not found in {keys_dir}")
            print("⚠️ Generating new keys (tokens will not work with MCP server)")
            print("⚠️ Make sure MCP server is running to generate shared keys")
            self._generate_keys()

    def _generate_keys(self):
        """Generate new RSA key pair (fallback only)"""
        self.private_key = rsa.generate_private_key(
            public_exponent=65537,
            key_size=2048,
            backend=default_backend()
        )
        self.public_key = self.private_key.public_key()

    def generate_token(self,
                      tenant_id: str,
                      user_id: str,
                      scopes: List[str],
                      expires_in_hours: int = 24) -> str:
        """Generate a JWT token for MCP server"""
        now = datetime.utcnow()
        payload = {
            "tenant_id": tenant_id,
            "user_id": user_id,
            "scopes": scopes,
            "iss": "mcp-server-demo",
            "aud": "mcp-server",
            "exp": now + timedelta(hours=expires_in_hours),
            "iat": now,
            "nbf": now
        }

        # Serialize private key to PEM format
        private_pem = self.private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption()
        )

        token = jwt.encode(
            payload,
            private_pem,
            algorithm="RS256"
        )
        return token

    def get_public_key_pem(self) -> str:
        """Get public key in PEM format"""
        public_pem = self.public_key.public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo
        )
        return public_pem.decode('utf-8')

    def decode_token(self, token: str) -> dict:
        """Decode and validate a JWT token"""
        try:
            public_pem = self.get_public_key_pem()
            decoded = jwt.decode(
                token,
                public_pem,
                algorithms=["RS256"],
                audience="mcp-server",
                issuer="mcp-server-demo"
            )
            return decoded
        except jwt.PyJWTError as e:
            raise ValueError(f"Invalid token: {str(e)}")


# Demo tenant IDs
DEMO_TENANTS = {
    "acme-corp": "11111111-1111-1111-1111-111111111111",
    "globex": "22222222-2222-2222-2222-222222222222",
    "initech": "33333333-3333-3333-3333-333333333333"
}

# Demo users for A2A
DEMO_USERS = {
    "demo-user-basic": {"budget": 10.0, "tier": "Basic"},
    "demo-user-pro": {"budget": 50.0, "tier": "Pro"},
    "demo-user-enterprise": {"budget": 200.0, "tier": "Enterprise"}
}
